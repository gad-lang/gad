module github.com/gad-lang/gad/giom

go 1.26.5

require (
	github.com/gad-lang/gad v0.0.4-0.20260717002044-7752b8fbcf85
	github.com/stretchr/testify v1.11.1
)

// giom lives in the gad repository as the ./giom submodule; build against the
// parent gad module in the working tree rather than a published version.
replace github.com/gad-lang/gad => ../

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
