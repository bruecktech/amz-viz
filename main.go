package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/autoscaling"
	"github.com/awslabs/aws-sdk-go/gen/cloudformation"
	"github.com/awslabs/aws-sdk-go/gen/ec2"
	"github.com/robfig/cron"
	"log"
	"net/http"
	"time"
)

var (
	creds        = aws.DetectCreds("", "", "")
	cli          = ec2.New(creds, "eu-west-1", nil)
	cfn          = cloudformation.New(creds, "eu-west-1", nil)
	asg          = autoscaling.New(creds, "eu-west-1", nil)
	dataVpc      = make(map[string]interface{})
	dataStack    = make(map[string]interface{})
	handlerVpc   = websocket.Handler(onConnectedVpc)
	handlerStack = websocket.Handler(onConnectedStack)
)

func resourcesByStackName(StackName string) []cloudformation.StackResource {
	resp, err := cfn.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{StackName: aws.String(StackName)})
	if err != nil {
		return nil
	}
	return resp.StackResources
}

func subnetsByVPCID(VPCID string) []ec2.Subnet {
	resp, err := cli.DescribeSubnets(&ec2.DescribeSubnetsRequest{Filters: []ec2.Filter{
		ec2.Filter{aws.String("vpc-id"), []string{VPCID}},
	}})
	if err != nil {
		return nil
	}
	return resp.Subnets
}

func instancesBySubnet(SubnetID string) []ec2.Instance {
	resp, err := cli.DescribeInstances(&ec2.DescribeInstancesRequest{Filters: []ec2.Filter{
		ec2.Filter{aws.String("subnet-id"), []string{SubnetID}},
	}})
	if err != nil {
		return nil
	}

	var instances []ec2.Instance

	for _, reservation := range resp.Reservations {
		instances = append(instances, reservation.Instances...)
	}
	return instances
}

func instancesByASG(AutoScalingGroupName string) []autoscaling.Instance {
	resp, err := asg.DescribeAutoScalingGroups(&autoscaling.AutoScalingGroupNamesType{AutoScalingGroupNames: []string{
		AutoScalingGroupName,
	}})
	if err != nil {
		return nil
	}

	return resp.AutoScalingGroups[0].Instances
}

func tagByKey(tags []ec2.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}

	return ""
}

func fetchDataVpc() {

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

	dataVpc["VPCs"] = vpcList
}

func fetchDataStack() {

	fmt.Println("Fetching data")

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

	type AutoScalingGroup struct {
		Name      string
		Instances []Instance
	}

	type Stack struct {
		Name              string
		Resources         []Resource
		Instances         []Instance
		AutoScalingGroups []AutoScalingGroup
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
		if resources != nil {
			for _, resource := range resources {

				switch *resource.ResourceType {
				case "AWS::EC2::Instance":
					tInstance := Instance{
						InstanceID: *resource.PhysicalResourceID,
						Name:       *resource.LogicalResourceID,
					}
					tStack.Instances = append(tStack.Instances, tInstance)
					continue
				case "AWS::AutoScaling::AutoScalingGroup":

					tAutoScalingGroup := AutoScalingGroup{
						Name: *resource.PhysicalResourceID,
					}

					instances := instancesByASG(*resource.PhysicalResourceID)

					if instances != nil {

						for _, instance := range instances {
							tInstance := Instance{
								InstanceID: *instance.InstanceID,
							}
							tAutoScalingGroup.Instances = append(tAutoScalingGroup.Instances, tInstance)
						}
						tStack.AutoScalingGroups = append(tStack.AutoScalingGroups, tAutoScalingGroup)
						continue
					}
				}

				tResource := Resource{
					Type:               *resource.ResourceType,
					LogicalResourceID:  *resource.LogicalResourceID,
					PhysicalResourceID: *resource.PhysicalResourceID,
				}

				tStack.Resources = append(tStack.Resources, tResource)
			}

			stackList = append(stackList, tStack)
		}
	}

	dataStack["Stacks"] = stackList
}

func onConnectedVpc(ws *websocket.Conn) {
	var err error

	for {
		json, _ := json.Marshal(dataVpc)
		if err = websocket.Message.Send(ws, string(json)); err != nil {
			fmt.Println("Can't send")
		}
		time.Sleep(10 * time.Second)
	}
}

func onConnectedStack(ws *websocket.Conn) {
	var err error

	for {
		json, _ := json.Marshal(dataStack)
		if err = websocket.Message.Send(ws, string(json)); err != nil {
			fmt.Println("Can't send")
		}
		time.Sleep(10 * time.Second)
	}
}

func main() {
	fetchDataVpc()
	fetchDataStack()

	c := cron.New()
	c.AddFunc("@every 1m", fetchDataVpc)
	c.AddFunc("@every 1m", fetchDataStack)
	c.Start()

	http.Handle("/vpc", handlerVpc)
	http.Handle("/stack", handlerStack)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("."))))

	log.Fatal(http.ListenAndServe(":8080", nil))

	c.Stop()
}
