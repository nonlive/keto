package aws

import (
	"reflect"
	"testing"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider/providers/aws/mocks"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func TestDescribeStacks(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	c := &Cloud{
		cf: mockCF,
	}

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{}).Return(
		&cloudformation.DescribeStacksOutput{}, awserr.New("ValidationError", "does not exist", nil),
	)

	res, err := c.describeStacks("")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Errorf("got number of results: %d; want: %d", len(res), 0)
	}

	mockCF.AssertExpectations(t)
}

func TestGetStacksByType(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	c := &Cloud{
		cf: mockCF,
	}

	stacks := []*cloudformation.Stack{
		{
			StackName: aws.String("master"),
			Tags: []*cloudformation.Tag{
				{
					Key:   aws.String(managedByKetoTagKey),
					Value: aws.String(managedByKetoTagValue),
				},
				{
					Key:   aws.String(stackTypeTagKey),
					Value: aws.String(masterPoolStackType),
				},
			},
		},
		{
			StackName: aws.String("compute-0"),
			Tags: []*cloudformation.Tag{
				{
					Key:   aws.String(managedByKetoTagKey),
					Value: aws.String(managedByKetoTagValue),
				},
				{
					Key:   aws.String(stackTypeTagKey),
					Value: aws.String(computePoolStackType),
				},
			},
		},
		{
			StackName: aws.String("compute-2"),
			Tags: []*cloudformation.Tag{
				{
					Key:   aws.String(stackTypeTagKey),
					Value: aws.String(computePoolStackType),
				},
			},
		},
	}

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{}).Return(
		&cloudformation.DescribeStacksOutput{Stacks: stacks}, nil)

	res, err := c.getStacksByType(computePoolStackType)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 1 {
		t.Fatal("must have received exactly one result")
	}
	if *res[0].StackName != "compute-0" {
		t.Errorf("got wrong stack: %q", *res[0].StackName)
	}

	mockCF.AssertExpectations(t)
}

func TestGetStackLabels(t *testing.T) {
	cases := []struct {
		name  string
		input []*cloudformation.Tag
		want  model.Labels
	}{
		{
			"label foo=bar, ignoring reserved labels",
			[]*cloudformation.Tag{
				{
					Key:   aws.String(managedByKetoTagKey),
					Value: aws.String(managedByKetoTagValue),
				},
				{
					Key:   aws.String(stackTypeTagKey),
					Value: aws.String(computePoolStackType),
				},
				{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
			},
			model.Labels{"foo": "bar"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := getStackLabels(&cloudformation.Stack{Tags: c.input})
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %#v; want %#v", got, c.want)
			}

		})
	}
}
