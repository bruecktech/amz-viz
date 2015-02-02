package main

//import "fmt"
import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/ec2"
	"github.com/gin-gonic/gin"
	//	"html/template"
)

var (
	creds = aws.DetectCreds("", "", "")
	cli   = ec2.New(creds, "eu-west-1", nil)
)

func subnetsByVPCID(VPCID string) []ec2.Subnet {
	resp, err := cli.DescribeSubnets(&ec2.DescribeSubnetsRequest{Filters: []ec2.Filter{ec2.Filter{aws.String("vpc-id"), []string{VPCID}}}})
	if err != nil {
		panic(err)
	}
	return resp.Subnets
}

func instancesBySubnet(SubnetID string) []ec2.Instance {
	resp, err := cli.DescribeInstances(&ec2.DescribeInstancesRequest{Filters: []ec2.Filter{ec2.Filter{aws.String("subnet-id"), []string{SubnetID}}}})
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

func viz(c *gin.Context) {
	type VPC struct {
		VPCID string
		Name  string
	}
	var vpcList []VPC

	type Subnet struct {
		SubnetID string
		Name     string
	}
	var subnetList []Subnet

	type Instance struct {
		InstanceID string
		Name       string
	}
	var instanceList []Instance

	resp, err := cli.DescribeVPCs(nil)
	if err != nil {
		panic(err)
	}
	for _, vpc := range resp.VPCs {
		vpcList = append(vpcList, VPC{
			VPCID: *vpc.VPCID,
			Name:  tagByKey(vpc.Tags, "Name")})

		subnets := subnetsByVPCID(*vpc.VPCID)
		for _, subnet := range subnets {
			subnetList = append(subnetList, Subnet{
				SubnetID: *subnet.SubnetID,
				Name:     tagByKey(subnet.Tags, "Name")})

			instances := instancesBySubnet(*subnet.SubnetID)

			for _, instance := range instances {
				instanceList = append(instanceList, Instance{
					InstanceID: *instance.InstanceID,
					Name:       tagByKey(instance.Tags, "Name")})

			}
		}
	}

	obj := gin.H{"vpcs": vpcList,
		"subnets":   subnetList,
		"instances": instanceList}
	c.HTML(200, "layout.tmpl", obj)

}

func main() {
	// Creates a gin router + logger and recovery (crash-free) middlewares
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/viz", viz)
	r.Static("/assets", "./assets")

	// Listen and server on 0.0.0.0:8080
	r.Run(":8080")
}
