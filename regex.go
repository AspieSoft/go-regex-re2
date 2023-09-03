package regex

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AspieSoft/go-regex-re2/common"
	"github.com/AspieSoft/go-syncterval"
	"github.com/AspieSoft/go-ttlcache"
	"github.com/pbnjay/memory"
)

type RE2 *regexp.Regexp

type Regexp struct {
	RE *regexp.Regexp
}

var regReReplaceQuoteRE2 *regexp.Regexp
var regReReplaceCommentRE2 *regexp.Regexp
var regReReplaceParamRE2 *regexp.Regexp

var regComplexSel *Regexp
var regEscape *Regexp

var regParamIndexCache *ttlcache.Cache[string, *regexp.Regexp] = ttlcache.New[string, *regexp.Regexp](2 * time.Hour, 1 * time.Hour)
var cache *ttlcache.Cache[string, *Regexp] = ttlcache.New[string, *Regexp](2 * time.Hour, 1 * time.Hour)

func init() {
	regReReplaceQuoteRE2 = regexp.MustCompile(`\\[\\']`)
	regReReplaceCommentRE2 = regexp.MustCompile(`\(\?\#.*?\)`)
	regReReplaceParamRE2 = regexp.MustCompile(`(\\|)(%\{[0-9]+\}|%[0-9])`)

	regComplexSel = Comp(`(\\|)\$([0-9]|\{[0-9]+\})`)
	regEscape = Comp(`[\\\^\$\.\|\?\*\+\(\)\[\]\{\}\%]`)

	go func(){
		// clear cache items older than 10 minutes if there are only 200MB of free memory
		syncterval.New(10 * time.Second, func() {
			if mem := common.FormatMemoryUsage(memory.FreeMemory()); mem < 200 && mem != 0 {
				cache.ClearEarly(10 * time.Minute)
				regParamIndexCache.ClearEarly(30 * time.Minute)
			}
		})
	}()
}

func setCache(re string, reg *Regexp) {
	cache.Set(re, reg)
}

func getCache(re string) (*Regexp, bool) {
	if val, ok := cache.Get(re); ok {
		return val, true
	}

	return &Regexp{}, false
}

// Comp compiles a regular expression and store it in the cache
func Comp(re string, params ...string) *Regexp {
	if strings.Contains(re, `\'`) {
		r := []byte(re)

		ind := regReReplaceQuoteRE2.FindAllIndex(r, -1)

		for i := len(ind) - 1; i >= 0; i-- {
			if r[ind[i][1]-1] == '\'' {
				r[ind[i][0]] = 0
				r[ind[i][1]-1] = '`'
			}
		}

		r = bytes.ReplaceAll(r, []byte{0}, []byte(""))
		re = string(r)
	}

	if strings.Contains(re, `(?#`) {
		re = regReReplaceCommentRE2.ReplaceAllString(re, ``)
	}

	for i, v := range params {
		var pRe string
		ind := strconv.Itoa(i+1)
		if len(ind) == 1 {
			pRe = `(\\|)(%\{`+ind+`\}|%`+ind+`)`
		} else {
			pRe = `(\\|)(%\{`+ind+`\})`
		}

		var pReC *regexp.Regexp
		if cache, ok := regParamIndexCache.Get(pRe); ok {
			pReC = cache
		}else{
			pReC = regexp.MustCompile(pRe)
		}

		re = string(pReC.ReplaceAllFunc([]byte(re), func(b []byte) []byte {
			if len(b) != 0 && b[0] == '\\' {
				return b
			}
			return []byte(Escape(v))
		}))
	}

	re = string(regReReplaceParamRE2.ReplaceAllFunc([]byte(re), func(b []byte) []byte {
		if len(b) != 0 && b[0] == '\\' {
			return b
		}
		return []byte{}
	}))

	if val, ok := getCache(re); ok {
		return val
	} else {
		reg := regexp.MustCompile(re)
		compRe := Regexp{RE: reg}

		go setCache(re, &compRe)
		return &compRe
	}
}

// CompTry tries to compile or returns an error
func CompTry(re string, params ...string) (*Regexp, error) {
	if strings.Contains(re, `\'`) {
		r := []byte(re)

		ind := regReReplaceQuoteRE2.FindAllIndex(r, -1)

		for i := len(ind) - 1; i >= 0; i-- {
			if r[ind[i][1]-1] == '\'' {
				r[ind[i][0]] = 0
				r[ind[i][1]-1] = '`'
			}
		}

		r = bytes.ReplaceAll(r, []byte{0}, []byte(""))
		re = string(r)
	}

	if strings.Contains(re, `(?#`) {
		re = regReReplaceCommentRE2.ReplaceAllString(re, ``)
	}

	for i, v := range params {
		var pRe string
		ind := strconv.Itoa(i+1)
		if len(ind) == 1 {
			pRe = `(\\|)(%\{`+ind+`\}|%`+ind+`)`
		} else {
			pRe = `(\\|)(%\{`+ind+`\})`
		}

		var pReC *regexp.Regexp
		if cache, ok := regParamIndexCache.Get(pRe); ok {
			pReC = cache
		}else{
			var err error
			pReC, err = regexp.Compile(pRe)
			if err != nil {
				return &Regexp{}, err
			}
		}

		re = string(pReC.ReplaceAllFunc([]byte(re), func(b []byte) []byte {
			if len(b) != 0 && b[0] == '\\' {
				return b
			}
			return []byte(Escape(v))
		}))
	}

	re = string(regReReplaceParamRE2.ReplaceAllFunc([]byte(re), func(b []byte) []byte {
		if len(b) != 0 && b[0] == '\\' {
			return b
		}
		return []byte{}
	}))

	if val, ok := getCache(re); ok {
		return val, nil
	} else {
		reg, err := regexp.Compile(re)
		if err != nil {
			return &Regexp{}, err
		}

		compRe := Regexp{RE: reg}

		go setCache(re, &compRe)
		return &compRe, nil
	}
}

// RepFunc replaces a string with the result of a function
// similar to JavaScript .replace(/re/, function(data){})
func (reg *Regexp) RepFunc(str []byte, rep func(data func(int) []byte) []byte, blank ...bool) []byte {
	ind := reg.RE.FindAllIndex(str, -1)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.FindAllSubmatch(v, -1)

		if len(blank) != 0 {
			gCache := map[int][]byte{}
			r := rep(func(g int) []byte {
				if v, ok := gCache[g]; ok {
					return v
				}
				v := []byte{}
				if len(m[0]) > g {
					v = m[0][g]
				}
				gCache[g] = v
				return v
			})

			if []byte(r) == nil {
				return []byte{}
			}
		} else {
			if trim == 0 {
				res = append(res, str[:pos[0]]...)
			} else {
				res = append(res, str[trim:pos[0]]...)
			}
			trim = pos[1]

			gCache := map[int][]byte{}
			r := rep(func(g int) []byte {
				if v, ok := gCache[g]; ok {
					return v
				}
				v := []byte{}
				if len(m[0]) > g {
					v = m[0][g]
				}
				gCache[g] = v
				return v
			})

			if []byte(r) == nil {
				res = append(res, str[trim:]...)
				return res
			}

			res = append(res, r...)
		}
	}

	if len(blank) != 0 {
		return []byte{}
	}

	res = append(res, str[trim:]...)

	return res
}

// RepStr replaces a string with another string
// note: this function is optimized for performance, and the replacement string does not accept replacements like $1
func (reg *Regexp) RepStr(str []byte, rep []byte) []byte {
	return reg.RE.ReplaceAll(str, rep)
}

// RepStrComp is a more complex version of the RepStr method
// this function will replace things in the result like $1 with your capture groups
// use $0 to use the full regex capture group
// use ${123} to use numbers with more than one digit
func (reg *Regexp) RepStrComp(str []byte, rep []byte) []byte {
	ind := reg.RE.FindAllIndex(str, -1)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.FindAllSubmatch(v, -1)

		if trim == 0 {
			res = append(res, str[:pos[0]]...)
		} else {
			res = append(res, str[trim:pos[0]]...)
		}
		trim = pos[1]

		r := regComplexSel.RepFunc(rep, func(data func(int) []byte) []byte {
			if len(data(1)) != 0 {
				return data(0)
			}
			n := data(2)
			if len(n) > 1 {
				n = n[1:len(n)-1]
			}
			if i, err := strconv.Atoi(string(n)); err == nil {
				if len(m[0]) > i {
					return m[0][i]
				}
			}
			return []byte{}
		})

		if r == nil {
			res = append(res, str[trim:]...)
			return res
		}

		res = append(res, r...)
		
	}

	res = append(res, str[trim:]...)

	return res
}

// Match returns true if a []byte matches a regex
func (reg *Regexp) Match(str []byte) bool {
	return reg.RE.Match(str)
}

// Split splits a string, and keeps capture groups
// Similar to JavaScript .split(/re/)
func (reg *Regexp) Split(str []byte) [][]byte {
	ind := reg.RE.FindAllIndex(str, -1)

	res := [][]byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.FindAllSubmatch(v, -1)

		if trim == 0 {
			res = append(res, str[:pos[0]])
		} else {
			res = append(res, str[trim:pos[0]])
		}
		trim = pos[1]

		for i := 1; i <= len(m[0]); i++ {
			g := m[0][i]
			if len(g) != 0 {
				res = append(res, m[0][i])
			}
		}
	}

	e := str[trim:]
	if len(e) != 0 {
		res = append(res, str[trim:])
	}

	return res
}


// IsValid will return true if a regex is valid and can compile
func IsValid(str []byte) bool {
	if _, err := regexp.Compile(string(str)); err == nil {
		return true
	}
	return false
}

// Escape will escape regex special chars
func Escape(re string) string {
	return string(regEscape.RepStrComp([]byte(re), []byte(`\$1`)))
}


// JoinBytes is an easy way to join multiple values into a single []byte
func JoinBytes(bytes ...interface{}) []byte {
	return common.JoinBytes(bytes...)
}