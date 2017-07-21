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

	"github.com/UKHomeOffice/keto/pkg/keto/util"
	"github.com/UKHomeOffice/keto/pkg/model"
)

func renderClusterInfraStackTemplate(c model.Cluster, vpcID string, networks []nodesNetwork) (string, error) {
	const (
		clusterInfraStackTemplate = `---
Description: "Kubernetes cluster '{{ .Cluster.Name }}' infra stack"

Resources:
  AssetsBucket:
    Type: AWS::S3::Bucket
    Properties:
      LifecycleConfiguration:
        Rules:
        - Id: expiry
          ExpirationInDays: '1'
          Status: Enabled

  MasterPoolSG:
    Type: "AWS::EC2::SecurityGroup"
    Properties:
      GroupDescription: "Kubernetes cluster {{ .Cluster.Name }} SG for master nodepool"
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
          Value: "keto-{{ .Cluster.Name }}-masterpool"
        - Key: KubernetesCluster
          Value: "{{ .Cluster.Name }}"

  # Allow traffic between master nodes.
  # TODO(vaijab): not all traffic needs to be allowed, maybe just etcd?
  MasterPoolAllTrafficSGIn:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref MasterPoolSG
      IpProtocol: -1
      SourceSecurityGroupId: !Ref MasterPoolSG
      FromPort: -1
      ToPort: -1

  MasterPoolComputeAPISGIn:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref MasterPoolSG
      IpProtocol: "6"
      SourceSecurityGroupId: !Ref ComputePoolSG
      FromPort: 443
      ToPort: 443

  ComputePoolSG:
    Type: "AWS::EC2::SecurityGroup"
    Properties:
      GroupDescription: "Kubernetes cluster {{ .Cluster.Name }} SG for compute nodepools"
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
          Value: "keto-{{ .Cluster.Name }}-computepool"
        - Key: KubernetesCluster
          Value: "{{ .Cluster.Name }}"

  # Allow traffic between all compute pools.
  # TODO(vaijab): would be nice to isolate different compute pools from each other.
  ComputePoolAllTrafficSGIn:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref ComputePoolSG
      IpProtocol: -1
      SourceSecurityGroupId: !Ref ComputePoolSG
      FromPort: -1
      ToPort: -1

  # Allow master nodes to talk to all compute pools.
  MasterPoolToComputePoolSG:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !Ref ComputePoolSG
      IpProtocol: "-1"
      SourceSecurityGroupId: !Ref MasterPoolSG
      # TODO(vaijab): not all ports need to be allowed.
      FromPort: "-1"
      ToPort: "-1"

{{ $clusterName := .Cluster.Name -}}
{{ range $_, $n := .Networks }}
  ENI{{ $n.NodeID }}:
    Type: "AWS::EC2::NetworkInterface"
    Properties:
      Description: "Kubernetes cluster {{ $clusterName }} master ENI"
      GroupSet:
        - !Ref MasterPoolSG
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
    Value: "{{ .VpcID }}"
    Export:
      Name:
        Fn::Sub: "${AWS::StackName}-VpcID"

  MasterPoolSG:
    Value: !Ref MasterPoolSG
    Export:
      Name:
        Fn::Sub: "${AWS::StackName}-MasterPoolSG"

  ComputePoolSG:
    Value: !Ref ComputePoolSG
    Export:
      Name:
        Fn::Sub: "${AWS::StackName}-ComputePoolSG"

  {{ .AssetsBucketNameOutputKey }}:
    Value: !Ref AssetsBucket
    Export:
      Name:
        Fn::Sub: "${AWS::StackName}-AssetsBucket"

  {{ .ClusterNameOutputKey }}:
    Value: "{{ .Cluster.Name }}"

  {{ .LabelsOutputKey }}:
    Value: "{{ .Labels }}"

  {{ .InternalClusterOutputKey }}:
    Value: "{{ .Cluster.Internal }}"

  {{ .StackTypeOutputKey }}:
    Value: "{{ .StackType }}"
`
	)

	data := struct {
		Cluster                   model.Cluster
		Networks                  []nodesNetwork
		VpcID                     string
		LabelsOutputKey           string
		Labels                    string
		ClusterNameOutputKey      string
		StackTypeOutputKey        string
		StackType                 string
		InternalClusterOutputKey  string
		AssetsBucketNameOutputKey string
	}{
		Cluster:                   c,
		Networks:                  networks,
		VpcID:                     vpcID,
		LabelsOutputKey:           labelsOutputKey,
		Labels:                    util.StringMapToKVs(c.Labels),
		ClusterNameOutputKey:      clusterNameOutputKey,
		StackTypeOutputKey:        stackTypeOutputKey,
		StackType:                 clusterInfraStackType,
		InternalClusterOutputKey:  internalClusterOutputKey,
		AssetsBucketNameOutputKey: assetsBucketNameOutputKey,
	}

	t := template.Must(template.New("cluster-infra-stack").Parse(clusterInfraStackTemplate))
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func renderELBStackTemplate(c model.Cluster, vpcID string) (string, error) {
	const (
		elbStackTemplate = `---
Description: "Kubernetes cluster '{{ .Cluster.Name }}' ELB stack"

Resources:
  ELBSG:
    Type: "AWS::EC2::SecurityGroup"
    Properties:
      GroupDescription: "Kubernetes cluster {{ .Cluster.Name }} SG for API ELB"
      VpcId: {{ .VpcID }}
      SecurityGroupIngress:
        - IpProtocol: "6"
          CidrIp: 0.0.0.0/0
          FromPort: "443"
          ToPort: "443"
      Tags:
        - Key: Name
          Value: "keto-{{ .Cluster.Name }}-kubeapi"

  # Allow ELB to talk to master node pool on 443/tcp
  ELBtoMasterPoolTrafficSG:
    Type: AWS::EC2::SecurityGroupIngress
    Properties:
      GroupId: !ImportValue "{{ .ClusterInfraStackName }}-MasterPoolSG"
      IpProtocol: "6"
      SourceSecurityGroupId: !Ref ELBSG
      FromPort: "443"
      ToPort: "443"

  ELB:
    Type: AWS::ElasticLoadBalancing::LoadBalancer
    Properties:
      CrossZone: true
      Subnets:
{{- range $index, $subnet := .Cluster.MasterPool.Networks }}
        - {{ $subnet }}
{{ end }}
      SecurityGroups:
        - !Ref ELBSG
      HealthCheck:
        Target: 'TCP:443'
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
          InstancePort: 443
          InstanceProtocol: TCP
      ConnectionSettings:
        IdleTimeout: 600
      Scheme: {{ if .Cluster.Internal }}"internal"{{ else }}"internet-facing"{{ end }}

{{ if ne .Cluster.DNSZone "" }}
  ELBDNS:
    Type: AWS::Route53::RecordSetGroup
    Properties:
      HostedZoneName: {{ .Cluster.DNSZone }}.
      RecordSets:
        - Name: kube-{{ .Cluster.Name }}.{{ .Cluster.DNSZone }}
          Type: A
          AliasTarget:
            HostedZoneId:
              'Fn::GetAtt':
                - ELB
                - CanonicalHostedZoneNameID
            DNSName:
              'Fn::GetAtt': [ ELB, {{ if .Cluster.Internal }}DNSName{{ else }}CanonicalHostedZoneName{{ end }} ]
{{ end }}

Outputs:
  ELB:
    Value: !Ref ELB
  {{ .ELBDNSOutputKey }}:
    {{ if ne .Cluster.DNSZone "" }}Value: kube-{{ .Cluster.Name }}.{{ .Cluster.DNSZone }}{{ else }}Value: {'Fn::GetAtt': [ ELB, {{ if .Cluster.Internal }}DNSName}{{ else }}CanonicalHostedZoneName{{ end }} ]}{{ end }}
`
	)

	// Make sure networks are always in the same order.
	sort.Strings(c.MasterPool.Networks)

	data := struct {
		Cluster                  model.Cluster
		VpcID                    string
		ClusterInfraStackName    string
		ClusterNameOutputKey     string
		StackTypeOutputKey       string
		StackType                string
		InternalClusterOutputKey string
		ELBDNSOutputKey          string
	}{
		Cluster: c,
		VpcID:   vpcID,
		ClusterInfraStackName:    makeClusterInfraStackName(c.Name),
		ClusterNameOutputKey:     clusterNameOutputKey,
		StackTypeOutputKey:       stackTypeOutputKey,
		StackType:                elbStackType,
		InternalClusterOutputKey: internalClusterOutputKey,
		ELBDNSOutputKey:          elbDNSOutputKey,
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
	amiID string,
	elbName string,
	assetsBucketName string,
	nodesPerSubnet map[string]int,
	kubeAPIURL string,
	stackName string,
) (string, error) {

	const (
		masterStackTemplate = `---
Description: "Kubernetes cluster '{{ .MasterPool.ClusterName }}' master nodepool stack"

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
      PolicyName: "kube-cluster-{{ .MasterPool.ClusterName }}-master-policy"
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
          - Resource:
              - Fn::Sub: "arn:aws:cloudformation:${AWS::Region}:${AWS::AccountId}:stack/{{ .StackName }}/*"
            Effect: Allow
            Action:
              - cloudformation:DescribeStacks
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

{{ $masterPool := .MasterPool -}}
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
          Value: "keto-{{ $masterPool.ClusterName }}-master"
          PropagateAtLaunch: true
        - Key: KubernetesCluster
          Value: "{{ $masterPool.ClusterName }}"
          PropagateAtLaunch: true
  LaunchConfiguration{{ rmdash $subnet }}:
    Type: AWS::AutoScaling::LaunchConfiguration
    Properties:
      AssociatePublicIpAddress: {{ if $masterPool.Internal }}false{{ else }}true{{ end }}
      IamInstanceProfile: !Ref InstanceProfile
      ImageId: "{{ $amiID }}"
      InstanceMonitoring: false
      InstanceType: "{{ $masterPool.MachineType }}"
      KeyName: "{{ $masterPool.SSHKey }}"
      SecurityGroups:
        - !ImportValue "{{ $clusterInfraStackName }}-MasterPoolSG"
      BlockDeviceMappings:
        - DeviceName: "/dev/xvda"
          Ebs:
            VolumeSize: "{{ $masterPool.DiskSize }}"
            DeleteOnTermination: true
            VolumeType: "gp2"
      UserData: {{ $userData }}
{{ end -}}

Outputs:
  {{ .AssetsBucketNameOutputKey }}:
    Value: "{{ .AssetsBucketName }}"

  {{ .ClusterNameOutputKey }}:
    Value: "{{ .MasterPool.ClusterName }}"

  {{ .PoolNameOutputKey }}:
    Value: "{{ .MasterPool.Name }}"

  {{ .CoreOSVersionOutputKey }}:
    Value: "{{ .MasterPool.CoreOSVersion }}"

  {{ .KubeAPIURLOutputKey }}:
    Value: "{{ .KubeAPIURL }}"

  {{ .MachineTypeOutputKey }}:
    Value: "{{ .MasterPool.MachineType }}"

  {{ .KubeVersionOutputKey }}:
    Value: "{{ .MasterPool.KubeVersion }}"

  {{ .DiskSizeOutputKey }}:
    Value: "{{ .MasterPool.DiskSize }}"

  {{ .LabelsOutputKey }}:
    Value: "{{ .Labels }}"

  {{ .InternalClusterOutputKey }}:
    Value: "{{ .MasterPool.Internal }}"

  {{ .StackTypeOutputKey }}:
    Value: "{{ .StackType }}"

  {{ .TaintsOutputKey }}:
    Value: "{{ .Taints }}"

  {{ .KubeletExtraArgsOutputKey }}:
    Value: "{{ .MasterPool.KubeletExtraArgs }}"

  {{ .APIServerExtraArgsOutputKey }}:
    Value: "{{ .MasterPool.APIServerExtraArgs }}"

  {{ .ControllerManagerExtraArgsOutputKey }}:
    Value: "{{ .MasterPool.ControllerManagerExtraArgs }}"

  {{ .SchedulerExtraArgsOutputKey }}:
    Value: "{{ .MasterPool.SchedulerExtraArgs }}"
`
	)

	// Make sure networks are always in the same order.
	sort.Strings(p.Networks)

	data := struct {
		MasterPool                          model.MasterPool
		ClusterInfraStackName               string
		StackName                           string
		AmiID                               string
		ELBName                             string
		UserData                            string
		NodesPerSubnet                      map[string]int
		KubeAPIURL                          string
		LabelsOutputKey                     string
		Labels                              string
		Taints                              string
		ClusterNameOutputKey                string
		PoolNameOutputKey                   string
		CoreOSVersionOutputKey              string
		StackTypeOutputKey                  string
		StackType                           string
		InternalClusterOutputKey            string
		AssetsBucketNameOutputKey           string
		AssetsBucketName                    string
		KubeAPIURLOutputKey                 string
		MachineTypeOutputKey                string
		KubeVersionOutputKey                string
		DiskSizeOutputKey                   string
		TaintsOutputKey                     string
		KubeletExtraArgsOutputKey           string
		APIServerExtraArgsOutputKey         string
		ControllerManagerExtraArgsOutputKey string
		SchedulerExtraArgsOutputKey         string
	}{
		MasterPool:                          p,
		ClusterInfraStackName:               makeClusterInfraStackName(p.ClusterName),
		StackName:                           stackName,
		AmiID:                               amiID,
		ELBName:                             elbName,
		UserData:                            base64.StdEncoding.EncodeToString(p.UserData),
		NodesPerSubnet:                      nodesPerSubnet,
		KubeAPIURL:                          kubeAPIURL,
		LabelsOutputKey:                     labelsOutputKey,
		Labels:                              util.StringMapToKVs(p.Labels),
		Taints:                              util.StringMapToKVs(p.Taints),
		ClusterNameOutputKey:                clusterNameOutputKey,
		CoreOSVersionOutputKey:              coreOSVersionOutputKey,
		PoolNameOutputKey:                   poolNameOutputKey,
		StackTypeOutputKey:                  stackTypeOutputKey,
		StackType:                           masterPoolStackType,
		InternalClusterOutputKey:            internalClusterOutputKey,
		AssetsBucketNameOutputKey:           assetsBucketNameOutputKey,
		AssetsBucketName:                    assetsBucketName,
		KubeAPIURLOutputKey:                 kubeAPIURLOutputKey,
		MachineTypeOutputKey:                machineTypeOutputKey,
		KubeVersionOutputKey:                kubeVersionOutputKey,
		DiskSizeOutputKey:                   diskSizeOutputKey,
		TaintsOutputKey:                     taintsOutputKey,
		KubeletExtraArgsOutputKey:           kubeletExtraArgsOutputKey,
		APIServerExtraArgsOutputKey:         apiServerExtraArgsOutputKey,
		ControllerManagerExtraArgsOutputKey: controllerManagerExtraArgsOutputKey,
		SchedulerExtraArgsOutputKey:         schedulerExtraArgsOutputKey,
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
	amiID string,
	kubeAPIURL string,
	stackName string,
) (string, error) {

	const (
		computeStackTemplate = `---
Description: "Kubernetes cluster '{{ .ComputePool.ClusterName }}' compute nodepool stack"

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
      PolicyName: "kube-cluster-{{ .ComputePool.ClusterName }}-compute-policy"
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
          - Resource:
              - Fn::Sub: "arn:aws:cloudformation:${AWS::Region}:${AWS::AccountId}:stack/{{ .StackName }}/*"
            Effect: Allow
            Action:
              - cloudformation:DescribeStacks

  ASG:
    Type: AWS::AutoScaling::AutoScalingGroup
    Properties:
      LaunchConfigurationName: !Ref LaunchConfiguration
      VPCZoneIdentifier:
{{- range $index, $subnet := .ComputePool.Networks }}
        - "{{ $subnet }}"
{{- end }}
      TerminationPolicies:
        - 'OldestInstance'
        - 'Default'
      MaxSize: 100
      MinSize: {{ .ComputePool.Size }}
      Tags:
        - Key: Name
          Value: "keto-{{ .ComputePool.ClusterName }}-{{ .ComputePool.Name }}"
          PropagateAtLaunch: true
        - Key: KubernetesCluster
          Value: "{{ .ComputePool.ClusterName }}"
          PropagateAtLaunch: true
  LaunchConfiguration:
    Type: AWS::AutoScaling::LaunchConfiguration
    Properties:
      AssociatePublicIpAddress: {{ if .ComputePool.Internal }}false{{ else }}true{{ end }}
      IamInstanceProfile: !Ref InstanceProfile
      ImageId: "{{ .AmiID }}"
      InstanceMonitoring: false
      InstanceType: "{{ .ComputePool.MachineType }}"
      KeyName: "{{ .ComputePool.SSHKey }}"
      SecurityGroups:
        - !ImportValue "{{ .ClusterInfraStackName }}-ComputePoolSG"
      BlockDeviceMappings:
        - DeviceName: "/dev/xvda"
          Ebs:
            VolumeSize: "{{ .ComputePool.DiskSize }}"
            DeleteOnTermination: true
            VolumeType: "gp2"
      UserData: {{ .UserData }}

Outputs:
  {{ .ClusterNameOutputKey }}:
    Value: "{{ .ComputePool.ClusterName }}"

  {{ .PoolNameOutputKey }}:
    Value: "{{ .ComputePool.Name }}"

  {{ .CoreOSVersionOutputKey }}:
    Value: "{{ .ComputePool.CoreOSVersion }}"

  {{ .KubeAPIURLOutputKey }}:
    Value: "{{ .KubeAPIURL }}"

  {{ .MachineTypeOutputKey }}:
    Value: "{{ .ComputePool.MachineType }}"

  {{ .KubeVersionOutputKey }}:
    Value: "{{ .ComputePool.KubeVersion }}"

  {{ .DiskSizeOutputKey }}:
    Value: "{{ .ComputePool.DiskSize }}"

  {{ .LabelsOutputKey }}:
    Value: "{{ .Labels }}"

  {{ .InternalClusterOutputKey }}:
    Value: "{{ .ComputePool.Internal }}"

  {{ .StackTypeOutputKey }}:
    Value: "{{ .StackType }}"

  {{ .TaintsOutputKey }}:
    Value: "{{ .Taints }}"

  {{ .KubeletExtraArgsOutputKey }}:
    Value: "{{ .ComputePool.KubeletExtraArgs }}"
`
	)

	// Make sure networks are always in the same order.
	sort.Strings(p.Networks)

	data := struct {
		ComputePool               model.ComputePool
		ClusterInfraStackName     string
		StackName                 string
		AmiID                     string
		UserData                  string
		KubeAPIURL                string
		LabelsOutputKey           string
		Labels                    string
		Taints                    string
		ClusterNameOutputKey      string
		PoolNameOutputKey         string
		CoreOSVersionOutputKey    string
		StackTypeOutputKey        string
		StackType                 string
		InternalClusterOutputKey  string
		KubeAPIURLOutputKey       string
		MachineTypeOutputKey      string
		KubeVersionOutputKey      string
		DiskSizeOutputKey         string
		TaintsOutputKey           string
		KubeletExtraArgsOutputKey string
	}{
		ComputePool:               p,
		ClusterInfraStackName:     makeClusterInfraStackName(p.ClusterName),
		StackName:                 stackName,
		AmiID:                     amiID,
		UserData:                  base64.StdEncoding.EncodeToString(p.UserData),
		KubeAPIURL:                kubeAPIURL,
		LabelsOutputKey:           labelsOutputKey,
		Labels:                    util.StringMapToKVs(p.Labels),
		Taints:                    util.StringMapToKVs(p.Taints),
		ClusterNameOutputKey:      clusterNameOutputKey,
		CoreOSVersionOutputKey:    coreOSVersionOutputKey,
		PoolNameOutputKey:         poolNameOutputKey,
		StackTypeOutputKey:        stackTypeOutputKey,
		StackType:                 computePoolStackType,
		InternalClusterOutputKey:  internalClusterOutputKey,
		KubeAPIURLOutputKey:       kubeAPIURLOutputKey,
		MachineTypeOutputKey:      machineTypeOutputKey,
		KubeVersionOutputKey:      kubeVersionOutputKey,
		DiskSizeOutputKey:         diskSizeOutputKey,
		TaintsOutputKey:           taintsOutputKey,
		KubeletExtraArgsOutputKey: kubeletExtraArgsOutputKey,
	}

	t := template.Must(template.New("compute-stack").Parse(computeStackTemplate))
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}

	return b.String(), nil
}
