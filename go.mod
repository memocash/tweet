module github.com/memocash/tweet

replace github.com/fallenstedt/twitter-stream => github.com/jchavannes/twitter-stream v0.0.0-20221222182917-b6a2ddd2363d

go 1.16

require (
	github.com/dghubble/go-twitter v0.0.0-20221024160433-0cc1e72ed6d8
	github.com/fallenstedt/twitter-stream v0.4.3-0.20221105030943-d555374f2c1a
	github.com/hasura/go-graphql-client v0.8.2-0.20221208014738-4e08b2d83631
	github.com/jchavannes/btcd v1.1.5-0.20230112162803-412def37b600
	github.com/jchavannes/jgo v0.0.0-20230124190857-0be599aa3e55
	github.com/memocash/index v0.1.1-0.20230131171049-b84f362d7013
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.14.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	golang.org/x/oauth2 v0.0.0-20221014153046-6fdb5e3db783
)
