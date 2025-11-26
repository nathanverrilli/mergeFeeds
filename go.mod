module mergeFeeds

go 1.25

require (
	github.com/nathanverrilli/denJson v0.1.0
	github.com/nathanverrilli/nlvMisc v0.1.0
	github.com/spf13/pflag v1.0.10
)

require github.com/shopspring/decimal v1.4.0 // indirect

// use local versions of these packages
// replace github.com/nathanverrilli/denJson v0.1.0 => ../denJson
// replace github.com/nathanverrilli/nlvMisc v0.1.0 => ../nlvMisc
