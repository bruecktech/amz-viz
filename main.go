package main

import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/cloudformation"
	"github.com/awslabs/aws-sdk-go/gen/ec2"
	"github.com/gin-gonic/gin"
)

var (
	creds = aws.DetectCreds("", "", "")
	cli   = ec2.New(creds, "eu-west-1", nil)
	cfn   = cloudformation.New(creds, "eu-west-1", nil)
)

func resourcesByStackName(StackName string) []cloudformation.StackResource {
	resp, err := cfn.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{StackName: aws.String(StackName)})
	if err != nil {
		panic(err)
	}
	return resp.StackResources
}

func subnetsByVPCID(VPCID string) []ec2.Subnet {
	resp, err := cli.DescribeSubnets(&ec2.DescribeSubnetsRequest{Filters: []ec2.Filter{
		ec2.Filter{aws.String("vpc-id"), []string{VPCID}},
	}})
	if err != nil {
		panic(err)
	}
	return resp.Subnets
}

func instancesBySubnet(SubnetID string) []ec2.Instance {
	resp, err := cli.DescribeInstances(&ec2.DescribeInstancesRequest{Filters: []ec2.Filter{
		ec2.Filter{aws.String("subnet-id"), []string{SubnetID}},
	}})
	if err != nil {
		panic(err)
	}

	var instances []ec2.Instance

	for _, reservation := range resp.Reservations {
		instances = append(instances, reservation.Instances...)
	}
	return instances
}

func tagByKey(tags []ec2.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}

	return ""
}

func vpc(c *gin.Context) {

	type Instance struct {
		InstanceID string
		Name       string
	}

	type Subnet struct {
		SubnetID  string
		Name      string
		Instances []Instance
	}

	type VPC struct {
		VPCID   string
		Name    string
		Subnets []Subnet
	}

	var vpcList []VPC

	resp, err := cli.DescribeVPCs(nil)
	if err != nil {
		panic(err)
	}
	for _, vpc := range resp.VPCs {
		tVPC := VPC{
			VPCID: *vpc.VPCID,
			Name:  tagByKey(vpc.Tags, "Name")}

		subnets := subnetsByVPCID(*vpc.VPCID)
		for _, subnet := range subnets {

			tSubnet := Subnet{
				SubnetID: *subnet.SubnetID,
				Name:     tagByKey(subnet.Tags, "Name")}

			instances := instancesBySubnet(*subnet.SubnetID)

			for _, instance := range instances {

				tInstance := Instance{
					InstanceID: *instance.InstanceID,
					Name:       tagByKey(instance.Tags, "Name")}

				tSubnet.Instances = append(tSubnet.Instances, tInstance)

			}

			tVPC.Subnets = append(tVPC.Subnets, tSubnet)
		}

		vpcList = append(vpcList, tVPC)
	}

	obj := gin.H{"VPCs": vpcList}
	c.HTML(200, "layout_vpc.tmpl", obj)

}

func stack(c *gin.Context) {

	type Instance struct {
		InstanceID string
		Name       string
	}

	type Subnet struct {
		SubnetID  string
		Name      string
		Instances []Instance
	}

	type Resource struct {
		Type               string
		LogicalResourceID  string
		PhysicalResourceID string
	}

	type Stack struct {
		Name      string
		Resources []Resource
	}

	var stackList []Stack

	resp, err := cfn.DescribeStacks(nil)
	if err != nil {
		panic(err)
	}
	for _, stack := range resp.Stacks {
		tStack := Stack{
			Name: *stack.StackName,
		}

		resources := resourcesByStackName(*stack.StackName)
		for _, resource := range resources {

			tResource := Resource{
				Type:               *resource.ResourceType,
				LogicalResourceID:  *resource.LogicalResourceID,
				PhysicalResourceID: *resource.PhysicalResourceID,
			}

			tStack.Resources = append(tStack.Resources, tResource)
		}

		stackList = append(stackList, tStack)
	}

	obj := gin.H{"Stacks": stackList}
	c.HTML(200, "layout_stack.tmpl", obj)

}

func main() {
	// Creates a gin router + logger and recovery (crash-free) middlewares
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/vpc", vpc)
	r.GET("/stack", stack)
	r.Static("/assets", "./assets")

	// Listen and server on 0.0.0.0:8080
	r.Run(":8080")
}
