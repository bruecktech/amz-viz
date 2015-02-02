package main

//import "fmt"
import "github.com/awslabs/aws-sdk-go/aws"
import "github.com/awslabs/aws-sdk-go/gen/ec2"
import "github.com/gin-gonic/gin"

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
  return "<html><head><title></title><link rel=\"stylesheet\" type=\"text/css\" href=\"/assets/style.css\" /></head><body>"
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

func viz(c *gin.Context){

  s := ""

  s += htmlHeader()

  resp, err := cli.DescribeVPCs(nil)
  if err != nil {
    panic(err)
  }
  for _, vpc := range resp.VPCs {
    s += divTagOpen("vpc", *vpc.VPCID + " (" + tagByKey(vpc.Tags, "Name") + ")")
    subnets := subnetsByVPCID(*vpc.VPCID)
    for _, subnet := range subnets {
      s += divTagOpen("subnet", *subnet.SubnetID + " (" + tagByKey(subnet.Tags, "Name") + ")")

      instances := instancesBySubnet(*subnet.SubnetID)
      for _, instance := range instances {
        s += divTagOpen("instance", *instance.InstanceID + " (" + tagByKey(instance.Tags, "Name") + ")")
        s += divTagClose()
      }
      s += divTagClose()
    }
    s += divTagClose()
  }
  s += htmlFooter()

  c.Data(200, "text/html", []byte(s))
}

func main() {
  // Creates a gin router + logger and recovery (crash-free) middlewares
  r := gin.Default()
  r.GET("/viz/", viz)
  r.Static("/assets", "./assets")

  // Listen and server on 0.0.0.0:8080
  r.Run(":8080")
}
