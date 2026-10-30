package commands

// stsGlobalRegion is used when no region is configured anywhere (saml2aws
// config, --region / SAML2AWS_REGION, AWS_REGION, or the shared AWS config).
//
// SDK v1 sent STS calls without a region to the global endpoint
// (https://sts.amazonaws.com, signed for us-east-1). SDK v2 instead fails with
// "Invalid Configuration: Missing Region". The pseudo-region "aws-global"
// makes SDK v2 resolve the same global endpoint, which keeps v1 behaviour.
//
// When a region IS configured, SDK v2 always uses the regional STS endpoint
// (e.g. https://sts.ap-northeast-1.amazonaws.com).
const stsGlobalRegion = "aws-global"
