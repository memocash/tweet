module github.com/memocash/tweet

replace github.com/gin-gonic/gin v1.6.3 => github.com/gin-gonic/gin v1.7.7

replace golang.org/x/net => golang.org/x/net v0.1.1-0.20221104162952-702349b0e862

go 1.16

require (
	github.com/aws/aws-sdk-go v1.44.264
	github.com/dghubble/go-twitter v0.0.0-20221024160433-0cc1e72ed6d8
	github.com/hasura/go-graphql-client v0.9.0
	github.com/jchavannes/btcd v1.1.5-0.20230112162803-412def37b600
	github.com/jchavannes/jgo v0.0.0-20230222214331-95b230651774
	github.com/memocash/index v0.2.0-alpha.0.0.20230719004923-ba2ac325c1bb
	github.com/n0madic/twitter-scraper v0.0.0-20230711213008-94503a2bc36c
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.14.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	golang.org/x/crypto v0.31.0
)
