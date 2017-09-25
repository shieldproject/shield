package config

var AWSHostToRegion = map[string]string{
	"s3.amazonaws.com":            "us-east-1",
	"s3-external-1.amazonaws.com": "us-east-1",

	"s3.us-east-2.amazonaws.com": "us-east-2",
	"s3-us-east-2.amazonaws.com": "us-east-2",

	"s3-us-west-1.amazonaws.com": "us-west-1",
	"s3-us-west-2.amazonaws.com": "us-west-2",

	"s3.ca-central-1.amazonaws.com": "ca-central-1",
	"s3-ca-central-1.amazonaws.com": "ca-central-1",

	"s3-eu-west-1.amazonaws.com": "eu-west-1",
	"s3-eu-west-2.amazonaws.com": "eu-west-2",

	"s3.eu-central-1.amazonaws.com": "eu-central-1",
	"s3-eu-central-1.amazonaws.com": "eu-central-1",

	"s3.ap-south-1.amazonaws.com": "ap-south-1",
	"s3-ap-south-1.amazonaws.com": "ap-south-1",

	"s3-ap-southeast-1.amazonaws.com": "ap-southeast-1",
	"s3-ap-southeast-2.amazonaws.com": "ap-southeast-2",

	"s3-ap-northeast-1.amazonaws.com": "ap-northeast-1",
	"s3.ap-northeast-2.amazonaws.com": "ap-northeast-2",
	"s3-ap-northeast-2.amazonaws.com": "ap-northeast-2",

	"s3-sa-east-1.amazonaws.com": "sa-east-1",

	"s3.cn-north-1.amazonaws.com.cn": "cn-north-1",
}

var multipartBlacklist = []string{
	"storage.googleapis.com",
}
