# Go Regex RE2

[![donation link](https://img.shields.io/badge/buy%20me%20a%20coffee-paypal-blue)](https://paypal.me/shaynejrtaylor?country.x=US&locale.x=en_US)

A High Performance PCRE Regex Package That Uses A Cache.

Simplifies the the go-pcre regex package.
After calling a regex, the compiled output gets cached to improve performance.

This package uses the [go-pcre](https://github.com/GRbit/go-pcre) package for better performance.

## Installation

```shell script
  go get github.com/AspieSoft/go-regex-re2
```

## Usage

```go

import (
  "github.com/AspieSoft/go-regex-re2"
)

// pre compile a regex into the cache
// this method also returns the compiled pcre.Regexp struct
regex.Comp(`re`)

// compile a regex and safely escape user input
regex.Comp(`re %1`, `this will be escaped .*`); // output: this will be escaped \.\*
regex.Comp(`re %1`, `hello \n world`); // output: hello \\n world (note: the \ was escaped, and the n is literal)
tree/v4.0.0
// use %n to reference a param
// use %{n} for param indexes with more than 1 digit
regex.Comp(`re %1 and %2 ... %{12}`, `param 1`, `param 2` ..., `param 12`);

// manually escape a string
regex.Escape(`(.*)? \$ \\$ \\\$ regex hack failed`)

// run a replace function (most advanced feature)
regex.Comp(`(?flags)re(capture group)`).RepFunc(myByteArray, func(data func(int) []byte) []byte {
  data(0) // get the string
  data(1) // get the first capture group

  return []byte("")

  // if the last option is true, returning nil will stop the loop early
  return nil
}, true /* optional: if true, will not process a return output */)

// run a simple light replace function
regex.Comp(`re`).RepStr(myByteArray, myReplacementByteArray)

// return a bool if a regex matches a byte array
regex.Comp(`re`).Match(myByteArray)

// split a byte array in a similar way to JavaScript
regex.Comp(`re|(keep this and split like in JavaScript)`).Split(myByteArray)

// a regex string is modified before compiling, to add a few other features
`use \' in place of ` + "`" + ` to make things easier`
`(?#This is a comment in regex)`

// an alias of *pcre.Regexp
regex.PCRE

// an alias of *regexp.Regexp
regex.RE2

// direct access to compiled *pcre.Regexp (or *regexp.Regexp if used)
regex.Comp("re").RE


// another helpful function
// this method makes it easier to return results to a regex function
regex.JoinBytes("string", []byte("byte array"), 10, 'c', data(2))

// the above method can be used in place of this one
append(append(append(append([]byte("string"), []byte("byte array")...), []byte(strconv.Itoa(10))...), 'c'), data(2)...)

```
