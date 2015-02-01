package main

import "fmt"
import "github.com/awslabs/aws-sdk-go/aws"
import "github.com/awslabs/aws-sdk-go/gen/ec2"

var creds = aws.DetectCreds("","","")
var cli = ec2.New(creds,"eu-west-1",nil)

func subnetsByVPCID(VPCID string) []ec2.Subnet {
  resp, err := cli.DescribeSubnets(&ec2.DescribeSubnetsRequest { Filters: []ec2.Filter{ ec2.Filter { aws.String("vpc-id"), []string{VPCID}}}})
  if err != nil {
    panic(err)
  }
  return resp.Subnets
}

func instancesBySubnet(SubnetID string) []ec2.Instance {
  resp, err := cli.DescribeInstances(&ec2.DescribeInstancesRequest { Filters: []ec2.Filter{ ec2.Filter { aws.String("subnet-id"), []string{SubnetID}}}})
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

func htmlHeader() string {
  return "<html><head><title></title><link rel=\"stylesheet\" type=\"text/css\" href=\"style.css\" /></head><body>"
}

func divTagOpen(class string, label string) string {
  return "<div class=\"" + class + "\">" + label
}

func divTagClose() string {
  return "</div>"
}

func htmlFooter() string {
  return "</body></html>"
}

func main() {
  fmt.Println(htmlHeader())
  resp, err := cli.DescribeVPCs(nil)
  if err != nil {
    panic(err)
  }
  for _, vpc := range resp.VPCs {
    //fmt.Println(*vpc.VPCID + " (" + tagByKey(vpc.Tags, "Name") + ")")
    fmt.Println(divTagOpen("vpc", *vpc.VPCID + " (" + tagByKey(vpc.Tags, "Name") + ")"))
    subnets := subnetsByVPCID(*vpc.VPCID)
    for _, subnet := range subnets {
      //fmt.Print("|-")
      //fmt.Println(*subnet.SubnetID + " (" + tagByKey(subnet.Tags, "Name") + ")")
      fmt.Println(divTagOpen("subnet", *subnet.SubnetID + " (" + tagByKey(subnet.Tags, "Name") + ")"))

      instances := instancesBySubnet(*subnet.SubnetID)
      for _, instance := range instances {
        //fmt.Print("  |-")
        //fmt.Println(*instance.InstanceID + " (" + tagByKey(instance.Tags, "Name") + ")")
        fmt.Println(divTagOpen("instance", *instance.InstanceID + " (" + tagByKey(instance.Tags, "Name") + ")"))
        fmt.Println(divTagClose())
      }
      fmt.Println(divTagClose())
    }
    fmt.Println(divTagClose())
  }
  fmt.Println(htmlFooter())
}
