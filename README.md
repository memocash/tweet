## Requirements

1. Index server: https://github.com/memocash/index
2. Twitter API key: https://developer.twitter.com/en/docs/basics/authentication/guides/access-tokens.html

## Example Config

Put in `./config.yaml`:

```yaml
BOT_SEED: hotel obvious agent lecture gadget evil jealous keen fragile before damp clarify
TWITTER_CREDS:
  USER_NAME: TwitterUser
  PASSWORD: password1234
```

## Build

```sh
go build
```

## Transfer Tweet History

*Must fund address first*

```sh
# ./tweet transfertweets <private_key> <twitter_handle>
./tweet transfertweets KyE5L74NaxjFvSdgLthaozgsudui1KVCj3DnXkZfcMxaR4uXLsE8 elonmusk
```

## Listen for New Tweets

```sh
# ./tweet getnewtweets <private_key> <twitter_handle>
./tweet getnewtweets KyE5L74NaxjFvSdgLthaozgsudui1KVCj3DnXkZfcMxaR4uXLsE8 elonmusk
```

## Mirror Bot

```sh
./tweet bot run
```

Then send memos with at least 5,000 sats in funds to bot address to control bot. You'll need more if you use the --history flag.
Some examples:

```text
CREATE TWITTER elonmusk
CREATE TWITTER elonmusk --history
```
