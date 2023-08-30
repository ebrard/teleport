/*
Copyright 2023 Gravitational, Inc.

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

package awsoidc

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gravitational/trace"
)

// CreateEC2ICERequest contains the required fields to create an AWS EC2 Instance Connect Endpoint.
type CreateEC2ICERequest struct {
	// Cluster is the Teleport Cluster Name.
	// Used to tag resources created in AWS.
	Cluster string

	// IntegrationName is the integration name.
	// Used to tag resources created in AWS.
	IntegrationName string

	// SubnetID is the Subnet where the Endpoint will be created.
	SubnetID string

	// SecurityGroupIDs is a list of SecurityGroups to assign to the Endpoint.
	// If not specified, the Endpoint will receive the default SG for the Subnet's VPC.
	SecurityGroupIDs []string

	// ResourceCreationTags is used to add tags when creating resources in AWS.
	// Defaults to:
	// - teleport.dev/cluster: <cluster>
	// - teleport.dev/origin: aws-oidc-integration
	// - teleport.dev/integration: <integrationName>
	ResourceCreationTags AWSTags
}

// CheckAndSetDefaults checks if the required fields are present.
func (req *CreateEC2ICERequest) CheckAndSetDefaults() error {
	if req.Cluster == "" {
		return trace.BadParameter("cluster is required")
	}

	if req.IntegrationName == "" {
		return trace.BadParameter("integration is required")
	}

	if req.SubnetID == "" {
		return trace.BadParameter("subnet id is required")
	}

	if len(req.ResourceCreationTags) == 0 {
		req.ResourceCreationTags = defaultResourceCreationTags(req.Cluster, req.IntegrationName)
	}

	return nil
}

// CreateEC2ICEResponse contains the newly created EC2 Instance Connect Endpoint name.
type CreateEC2ICEResponse struct {
	// Name is the Endpoint ID.
	Name string
}

// CreateEC2ICE describes the required methods to List EC2 Instances using a 3rd Party API.
type CreateEC2ICEClient interface {
	// CreateInstanceConnectEndpoint creates an EC2 Instance Connect Endpoint. An EC2 Instance Connect Endpoint
	// allows you to connect to an instance, without requiring the instance to have a
	// public IPv4 address. For more information, see Connect to your instances
	// without requiring a public IPv4 address using EC2 Instance Connect Endpoint (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/Connect-using-EC2-Instance-Connect-Endpoint.html)
	// in the Amazon EC2 User Guide.
	CreateInstanceConnectEndpoint(ctx context.Context, params *ec2.CreateInstanceConnectEndpointInput, optFns ...func(*ec2.Options)) (*ec2.CreateInstanceConnectEndpointOutput, error)
}

type defaultCreateEC2ICEClient struct {
	*ec2.Client
}

// NewCreateEC2ICEClient creates a new CreateEC2ICEClient using a AWSClientRequest.
func NewCreateEC2ICEClient(ctx context.Context, req *AWSClientRequest) (CreateEC2ICEClient, error) {
	ec2Client, err := newEC2Client(ctx, req)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &defaultCreateEC2ICEClient{
		Client: ec2Client,
	}, nil
}

// CreateEC2ICE calls the following AWS API:
// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateInstanceConnectEndpoint.html
// It creates an EC2 Instance Connect Endpoint using the provided Subnet and Security Group IDs.
func CreateEC2ICE(ctx context.Context, clt CreateEC2ICEClient, req CreateEC2ICERequest) (*CreateEC2ICEResponse, error) {
	if err := req.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	createEC2ICEInput := &ec2.CreateInstanceConnectEndpointInput{
		SubnetId:         &req.SubnetID,
		SecurityGroupIds: req.SecurityGroupIDs,
		TagSpecifications: []ec2types.TagSpecification{{
			ResourceType: ec2types.ResourceTypeInstanceConnectEndpoint,
			Tags:         req.ResourceCreationTags.ToEC2Tags(),
		}},
	}

	ec2ICEndpoint, err := clt.CreateInstanceConnectEndpoint(ctx, createEC2ICEInput)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	endpointName := "unknown"
	if ec2ICEndpoint.InstanceConnectEndpoint != nil {
		endpointName = aws.ToString(ec2ICEndpoint.InstanceConnectEndpoint.InstanceConnectEndpointId)
	}

	return &CreateEC2ICEResponse{
		Name: endpointName,
	}, nil
}