module github.com/memocash/tweet

replace github.com/fallenstedt/twitter-stream => github.com/jchavannes/twitter-stream v0.0.0-20221222182917-b6a2ddd2363d

replace github.com/gin-gonic/gin v1.6.3 => github.com/gin-gonic/gin v1.7.7

replace golang.org/x/net => golang.org/x/net v0.1.1-0.20221104162952-702349b0e862

go 1.16

require (
	github.com/dghubble/go-twitter v0.0.0-20221024160433-0cc1e72ed6d8
	github.com/fallenstedt/twitter-stream v0.4.3-0.20221105030943-d555374f2c1a
	github.com/hasura/go-graphql-client v0.9.0
	github.com/jchavannes/btcd v1.1.5-0.20230112162803-412def37b600
	github.com/jchavannes/jgo v0.0.0-20230222214331-95b230651774
	github.com/memocash/index v0.2.0-alpha.0.0.20230323182811-fd9337017bac
	github.com/n0madic/twitter-scraper v0.0.0-20230429212535-64a2f495ca7c // indirect
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.14.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	golang.org/x/crypto v0.6.0
	golang.org/x/oauth2 v0.5.0
)
