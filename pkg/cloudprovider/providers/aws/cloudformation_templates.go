/*
Copyright 2017 The Keto Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"bytes"
	"encoding/base64"
	"sort"
	"strings"
	"text/template"

	"github.com/UKHomeOffice/keto/pkg/model"
)

func renderClusterInfraStackTemplate(clusterName, vpcID string, networks []nodesNetwork) (string, error) {
	const (
		clusterInfraStackTemplate = `---
Description: "Kubernetes cluster '{{ .ClusterName }}' infra stack"

Resources:
  AssetsBucket:
    Type: AWS::S3::Bucket

  MasterNodePoolSG:
    Type: "AWS::EC2::SecurityGroup"
    Properties:
      GroupDescription: "Kubernetes cluster {{ .ClusterName }} SG for master nodepool"
      VpcId: {{ .VpcID }}
      SecurityGroupIngress:
        - IpProtocol: "6"
          CidrIp: 0.0.0.0/0
          FromPort: "22"
          ToPort: "22"
      SecurityGroupEgress:
        - IpProtocol: -1
          CidrIp: 0.0.0.0/0
          FromPort: -1
          ToPort: -1
      Tags:
        - Key: Name
          Value: "keto-{{ .ClusterName }}-masterpool"
        - Key: KubernetesCluster
          Value: "{{ .ClusterName }}"

  # Allow traffic between master nodes.
  # TODO(vaijab): not all traffic needs to be allowed, maybe just etcd?
  MasterNodePoolAllTrafficSGIn:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref MasterNodePoolSG
      IpProtocol: -1
      SourceSecurityGroupId: !Ref MasterNodePoolSG
      FromPort: -1
      ToPort: -1

  MasterNodePoolComputeAPISGIn:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref MasterNodePoolSG
      IpProtocol: "6"
      SourceSecurityGroupId: !Ref ComputeNodePoolSG
      FromPort: 6443
      ToPort: 6443

  ComputeNodePoolSG:
    Type: "AWS::EC2::SecurityGroup"
    Properties:
      GroupDescription: "Kubernetes cluster {{ .ClusterName }} SG for compute nodepools"
      VpcId: {{ .VpcID }}
      SecurityGroupIngress:
        - IpProtocol: "6"
          CidrIp: 0.0.0.0/0
          FromPort: "22"
          ToPort: "22"
      SecurityGroupEgress:
        - IpProtocol: -1
          CidrIp: 0.0.0.0/0
          FromPort: -1
          ToPort: -1
      Tags:
        - Key: Name
          Value: "keto-{{ .ClusterName }}-computepool"
        - Key: KubernetesCluster
          Value: "{{ .ClusterName }}"

  # Allow traffic between all compute pools.
  # TODO(vaijab): would be nice to isolate different compute pools from each other.
  ComputeNodePoolAllTrafficSGIn:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref ComputeNodePoolSG
      IpProtocol: -1
      SourceSecurityGroupId: !Ref ComputeNodePoolSG
      FromPort: -1
      ToPort: -1

  # Allow master nodes to talk to all compute pools.
  MasterNodePoolToComputeNodePoolSG:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref ComputeNodePoolSG
      IpProtocol: "-1"
      SourceSecurityGroupId: !Ref MasterNodePoolSG
      # TODO(vaijab): not all ports need to be allowed.
      FromPort: "-1"
      ToPort: "-1"

{{ $clusterName := .ClusterName -}}
{{ range $_, $n := .Networks }}
  ENI{{ $n.NodeID }}:
    Type: "AWS::EC2::NetworkInterface"
    Properties:
      Description: "Kubernetes cluster {{ $clusterName }} master ENI"
      GroupSet:
        - !Ref MasterNodePoolSG
      SourceDestCheck: false
      SubnetId: "{{ $n.Subnet }}"
      Tags:
        # Required for smilodon
        - Key: "NodeID"
          Value: "{{ $n.NodeID }}"
        - Key: Name
          Value: "keto-{{ $clusterName }}-eni{{ $n.NodeID }}"

  Volume{{ $n.NodeID }}:
    Type: AWS::EC2::Volume
    Properties:
      Encrypted: true
      Size: 10
      VolumeType: gp2
      AvailabilityZone: {{ $n.AvailabilityZone }}
      Tags:
        # Required for smilodon
        - Key: NodeID
          Value: "{{ $n.NodeID }}"
        - Key: Name
          Value: "keto-{{ $clusterName }}-volume{{ $n.NodeID }}"
  {{ end }}

Outputs:
  VpcID:
    Value: {{ .VpcID }}
    Export:
      Name:
        Fn::Sub: "${AWS::StackName}-VpcID"
  AssetsBucket:
    Value: !Ref AssetsBucket
    Export:
      Name:
        Fn::Sub: "${AWS::StackName}-AssetsBucket"
  MasterNodePoolSG:
    Value: !Ref MasterNodePoolSG
    Export:
      Name:
        Fn::Sub: "${AWS::StackName}-MasterNodePoolSG"
  ComputeNodePoolSG:
    Value: !Ref ComputeNodePoolSG
    Export:
      Name:
        Fn::Sub: "${AWS::StackName}-ComputeNodePoolSG"
`
	)

	data := struct {
		ClusterName string
		Networks    []nodesNetwork
		VpcID       string
	}{
		ClusterName: clusterName,
		Networks:    networks,
		VpcID:       vpcID,
	}

	t := template.Must(template.New("cluster-infra-stack").Parse(clusterInfraStackTemplate))
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func renderELBStackTemplate(p model.MasterPool, vpcID, clusterInfraStackName string) (string, error) {
	const (
		elbStackTemplate = `---
Description: "Kubernetes cluster '{{ .ClusterName }}' ELB stack"

Resources:
  ELBSG:
    Type: "AWS::EC2::SecurityGroup"
    Properties:
      GroupDescription: "Kubernetes cluster {{ .ClusterName }} SG for API ELB"
      VpcId: {{ .VpcID }}
      SecurityGroupIngress:
        - IpProtocol: "6"
          CidrIp: 0.0.0.0/0
          FromPort: "443"
          ToPort: "443"
        - IpProtocol: "6"
          CidrIp: 0.0.0.0/0
          FromPort: "6443"
          ToPort: "6443"
      Tags:
        - Key: Name
          Value: "keto-{{ .ClusterName }}-kubeapi"

  # Allow ELB to talk to master node pool on 6443/tcp
  ELBtoMasterNodePoolTrafficSG:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !ImportValue "{{ .ClusterInfraStackName }}-MasterNodePoolSG"
      IpProtocol: "6"
      SourceSecurityGroupId: !Ref ELBSG
      FromPort: "6443"
      ToPort: "6443"

  ELB:
    Type: AWS::ElasticLoadBalancing::LoadBalancer
    Properties:
      CrossZone: true
      Subnets:
{{- range $index, $subnet := .Networks }}
        - {{ $subnet }}
{{ end }}
      SecurityGroups:
        - !Ref ELBSG
      HealthCheck:
        Target: 'TCP:6443'
        HealthyThreshold: 2
        Interval: 10
        Timeout: 5
        UnhealthyThreshold: 2
      ConnectionDrainingPolicy:
        Enabled: true
        Timeout: 30
      Listeners:
        - LoadBalancerPort: 443
          Protocol: TCP
          InstancePort: 6443
          InstanceProtocol: TCP
        - LoadBalancerPort: 6443
          Protocol: TCP
          InstancePort: 6443
          InstanceProtocol: TCP
      ConnectionSettings:
        IdleTimeout: 600

Outputs:
  ELB:
    Value: !Ref ELB
`
	)

	// Make sure networks are always in the same order.
	sort.Strings(p.Networks)

	data := struct {
		ClusterName           string
		Networks              []string
		VpcID                 string
		ClusterInfraStackName string
	}{
		ClusterName: p.ClusterName,
		Networks:    p.KubeAPINetworks,
		VpcID:       vpcID,
		ClusterInfraStackName: clusterInfraStackName,
	}

	t := template.Must(template.New("elb-stack").Parse(elbStackTemplate))
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func renderMasterStackTemplate(
	p model.MasterPool,
	clusterInfraStackName string,
	amiID string,
	elbName string,
	assetsBucketName string,
	nodesPerSubnet map[string]int,
) (string, error) {

	const (
		masterStackTemplate = `---
Description: "Kubernetes cluster '{{ .MasterNodePool.ClusterName }}' master nodepool stack"

Resources:
  InstanceRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Statement:
          - Principal:
              Service:
                - "ec2.amazonaws.com"
            Effect: Allow
            Action:
              - "sts:AssumeRole"
      Path: /

  InstanceProfile:
    Type: AWS::IAM::InstanceProfile
    Properties:
      Roles:
        - !Ref InstanceRole
      Path: /

  RolePolicies:
    Type: AWS::IAM::Policy
    Properties:
      PolicyName: "kube-cluster-{{ .MasterNodePool.ClusterName }}-master-policy"
      Roles:
        - !Ref InstanceRole
      PolicyDocument:
        Statement:
          - Resource: "*"
            Effect: Allow
            Action:
              - autoscaling:DescribeAutoScalingGroups
              - ec2:CreateTags
              - ec2:DescribeTags
              - ec2:DescribeInstances
          - Resource: "arn:aws:s3:::{{ .AssetsBucketName }}"
            Effect: Allow
            Action:
              - "s3:List*"
          - Resource: "arn:aws:s3:::{{ .AssetsBucketName }}/*"
            Effect: Allow
            Action:
              - "s3:Get*"
          - Resource: "*"
            Effect: Allow
            Action:
              - ec2:AttachNetworkInterface
              - ec2:AttachVolume
              - ec2:AuthorizeSecurityGroupEgress
              - ec2:AuthorizeSecurityGroupIngress
              - ec2:CreateRoute
              - ec2:CreateSecurityGroup
              - ec2:CreateTags
              - ec2:CreateVolume
              - ec2:DeleteRoute
              - ec2:DeleteSecurityGroup
              - ec2:DeleteVolume
              - ec2:DescribeInstances
              - ec2:DescribeNetworkInterfaces
              - ec2:DescribeRouteTables
              - ec2:DescribeRouteTables
              - ec2:DescribeSecurityGroups
              - ec2:DescribeSubnets
              - ec2:DescribeTags
              - ec2:DescribeVolumes
              - ec2:DescribeVpcs
              - ec2:DetachNetworkInterface
              - ec2:DetachVolume
              - ec2:ModifyInstanceAttribute
              - ec2:ModifyNetworkInterfaceAttribute
              - ec2:RevokeSecurityGroupEgress
              - ec2:RevokeSecurityGroupIngress
              - elasticloadbalancing:ConfigureHealthCheck
              - elasticloadbalancing:Create*
              - elasticloadbalancing:Delete*
              - elasticloadbalancing:DeregisterInstancesFromLoadBalancer
              - elasticloadbalancing:DescribeLoadBalancerAttributes
              - elasticloadbalancing:DescribeLoadBalancers
              - elasticloadbalancing:ModifyLoadBalancerAttributes
              - elasticloadbalancing:RegisterInstancesWithLoadBalancer
              - elasticloadbalancing:SetLoadBalancerPoliciesForBackendServer

{{ $masterNodePool := .MasterNodePool -}}
{{ $userData := .UserData -}}
{{ $amiID := .AmiID -}}
{{ $elbName := .ELBName -}}
{{ $clusterInfraStackName := .ClusterInfraStackName -}}
{{ range $subnet, $num := .NodesPerSubnet }}
  ASG{{ rmdash $subnet }}:
    Type: AWS::AutoScaling::AutoScalingGroup
    Properties:
      LaunchConfigurationName: !Ref LaunchConfiguration{{ rmdash $subnet }}
      VPCZoneIdentifier:
        - "{{ $subnet }}"
      LoadBalancerNames:
        - "{{ $elbName }}"
      TerminationPolicies:
        - 'OldestInstance'
        - 'Default'
      MaxSize: {{ $num }}
      MinSize: {{ $num }}
      Tags:
        - Key: Name
          Value: "keto-{{ $masterNodePool.ClusterName }}-master"
          PropagateAtLaunch: true
        - Key: KubernetesCluster
          Value: "{{ $masterNodePool.ClusterName }}"
          PropagateAtLaunch: true
  LaunchConfiguration{{ rmdash $subnet }}:
    Type: AWS::AutoScaling::LaunchConfiguration
    Properties:
      AssociatePublicIpAddress: true
      IamInstanceProfile: !Ref InstanceProfile
      ImageId: "{{ $amiID }}"
      InstanceMonitoring: false
      InstanceType: "{{ $masterNodePool.MachineType }}"
      KeyName: "{{ $masterNodePool.SSHKey }}"
      SecurityGroups:
        - !ImportValue "{{ $clusterInfraStackName }}-MasterNodePoolSG"
      BlockDeviceMappings:
        - DeviceName: "/dev/xvda"
          Ebs:
            VolumeSize: "{{ $masterNodePool.DiskSize }}"
            DeleteOnTermination: true
            VolumeType: "gp2"
      UserData: {{ $userData }}
  {{ end -}}
`
	)

	// Make sure networks are always in the same order.
	sort.Strings(p.Networks)

	data := struct {
		MasterNodePool        model.MasterPool
		ClusterInfraStackName string
		AmiID                 string
		ELBName               string
		UserData              string
		AssetsBucketName      string
		NodesPerSubnet        map[string]int
	}{
		MasterNodePool:        p,
		ClusterInfraStackName: clusterInfraStackName,
		AmiID:            amiID,
		ELBName:          elbName,
		UserData:         base64.StdEncoding.EncodeToString(p.UserData),
		AssetsBucketName: assetsBucketName,
		NodesPerSubnet:   nodesPerSubnet,
	}

	funcMap := template.FuncMap{
		// Deletes dashes from a string.
		"rmdash": func(s string) string {
			return strings.Replace(s, "-", "", -1)
		},
	}

	t := template.Must(template.New("master-stack").Funcs(funcMap).Parse(masterStackTemplate))
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func renderComputeStackTemplate(
	p model.ComputePool,
	clusterInfraStackName string,
	amiID string,
) (string, error) {

	const (
		computeStackTemplate = `---
Description: "Kubernetes cluster '{{ .ComputeNodePool.ClusterName }}' compute nodepool stack"

Resources:
  InstanceRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Statement:
          - Principal:
              Service:
                - "ec2.amazonaws.com"
            Effect: Allow
            Action:
              - "sts:AssumeRole"
      Path: /

  InstanceProfile:
    Type: AWS::IAM::InstanceProfile
    Properties:
      Roles:
        - !Ref InstanceRole
      Path: /

  RolePolicies:
    Type: AWS::IAM::Policy
    Properties:
      PolicyName: "kube-cluster-{{ .ComputeNodePool.ClusterName }}-compute-policy"
      Roles:
        - !Ref InstanceRole
      PolicyDocument:
        Statement:
          - Resource: "*"
            Effect: Allow
            Action:
              - ec2:CreateTags
              - ec2:DescribeInstances
              - ec2:DescribeTags
              - ec2:DescribeVpcs

  ASG:
    Type: AWS::AutoScaling::AutoScalingGroup
    Properties:
      LaunchConfigurationName: !Ref LaunchConfiguration
      VPCZoneIdentifier:
{{- range $index, $subnet := .ComputeNodePool.Networks }}
        - "{{ $subnet }}"
{{- end }}
      TerminationPolicies:
        - 'OldestInstance'
        - 'Default'
      MaxSize: 100
      MinSize: {{ .ComputeNodePool.Size }}
      Tags:
        - Key: Name
          Value: "keto-{{ .ComputeNodePool.ClusterName }}-compute"
          PropagateAtLaunch: true
        - Key: KubernetesCluster
          Value: "{{ .ComputeNodePool.ClusterName }}"
          PropagateAtLaunch: true
  LaunchConfiguration:
    Type: AWS::AutoScaling::LaunchConfiguration
    Properties:
      AssociatePublicIpAddress: true
      IamInstanceProfile: !Ref InstanceProfile
      ImageId: "{{ .AmiID }}"
      InstanceMonitoring: false
      InstanceType: "{{ .ComputeNodePool.MachineType }}"
      KeyName: "{{ .ComputeNodePool.SSHKey }}"
      SecurityGroups:
        - !ImportValue "{{ .ClusterInfraStackName }}-ComputeNodePoolSG"
      BlockDeviceMappings:
        - DeviceName: "/dev/xvda"
          Ebs:
            VolumeSize: "{{ .ComputeNodePool.DiskSize }}"
            DeleteOnTermination: true
            VolumeType: "gp2"
      UserData: {{ .UserData }}
`
	)

	// Make sure networks are always in the same order.
	sort.Strings(p.Networks)

	data := struct {
		ComputeNodePool       model.ComputePool
		ClusterInfraStackName string
		AmiID                 string
		UserData              string
	}{
		ComputeNodePool:       p,
		ClusterInfraStackName: clusterInfraStackName,
		AmiID:    amiID,
		UserData: base64.StdEncoding.EncodeToString(p.UserData),
	}

	t := template.Must(template.New("compute-stack").Parse(computeStackTemplate))
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}

	return b.String(), nil
}
