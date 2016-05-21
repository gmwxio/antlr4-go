// This implementation of {@link TokenStream} loads tokens from a
// {@link TokenSource} on-demand, and places the tokens in a buffer to provide
// access to any previous token by index.
//
// <p>
// This token stream ignores the value of {@link Token//getChannel}. If your
// parser requires the token stream filter tokens to only those on a particular
// channel, such as {@link Token//DEFAULT_CHANNEL} or
// {@link Token//HIDDEN_CHANNEL}, use a filtering token stream such a
// {@link CommonTokenStream}.</p>

package antlr

import (
	"fmt"
	"strconv"
)

type CommonTokenStream struct {
	tokenSource TokenSource

	tokens     []Token
	index      int
	fetchedEOF bool
	channel    int
}

func NewCommonTokenStream(lexer Lexer, channel int) *CommonTokenStream {

	ts := new(CommonTokenStream)

	// The {@link TokenSource} from which tokens for bt stream are fetched.
	ts.tokenSource = lexer

	// A collection of all tokens fetched from the token source. The list is
	// considered a complete view of the input once {@link //fetchedEOF} is set
	// to {@code true}.
	ts.tokens = make([]Token, 0)

	// The index into {@link //tokens} of the current token (next token to
	// {@link //consume}). {@link //tokens}{@code [}{@link //p}{@code ]} should
	// be
	// {@link //LT LT(1)}.
	//
	// <p>This field is set to -1 when the stream is first constructed or when
	// {@link //SetTokenSource} is called, indicating that the first token has
	// not yet been fetched from the token source. For additional information,
	// see the documentation of {@link IntStream} for a description of
	// Initializing Methods.</p>
	ts.index = -1

	// Indicates whether the {@link Token//EOF} token has been fetched from
	// {@link //tokenSource} and added to {@link //tokens}. This field improves
	// performance for the following cases:
	//
	// <ul>
	// <li>{@link //consume}: The lookahead check in {@link //consume} to
	// prevent
	// consuming the EOF symbol is optimized by checking the values of
	// {@link //fetchedEOF} and {@link //p} instead of calling {@link
	// //LA}.</li>
	// <li>{@link //fetch}: The check to prevent adding multiple EOF symbols
	// into
	// {@link //tokens} is trivial with bt field.</li>
	// <ul>
	ts.fetchedEOF = false

	ts.channel = channel

	return ts
}

func (c *CommonTokenStream) GetAllTokens() []Token {
	return c.tokens
}

func (c *CommonTokenStream) Mark() int {
	return 0
}

func (c *CommonTokenStream) Release(marker int) {
	// no resources to release
}

func (c *CommonTokenStream) reset() {
	c.Seek(0)
}

func (c *CommonTokenStream) Seek(index int) {
	c.lazyInit()
	c.index = c.adjustSeekIndex(index)
}

func (c *CommonTokenStream) Get(index int) Token {
	c.lazyInit()
	return c.tokens[index]
}

func (c *CommonTokenStream) Consume() {
	var SkipEofCheck = false
	if c.index >= 0 {
		if c.fetchedEOF {
			// the last token in tokens is EOF. Skip check if p indexes any
			// fetched token except the last.
			SkipEofCheck = c.index < len(c.tokens)-1
		} else {
			// no EOF token in tokens. Skip check if p indexes a fetched token.
			SkipEofCheck = c.index < len(c.tokens)
		}
	} else {
		// not yet initialized
		SkipEofCheck = false
	}

	if PortDebug {
		fmt.Println("Consume 1")
	}
	if !SkipEofCheck && c.LA(1) == TokenEOF {
		panic("cannot consume EOF")
	}
	if c.Sync(c.index + 1) {
		if PortDebug {
			fmt.Println("Consume 2")
		}
		c.index = c.adjustSeekIndex(c.index + 1)
	}
}

// Make sure index {@code i} in tokens has a token.
//
// @return {@code true} if a token is located at index {@code i}, otherwise
// {@code false}.
// @see //Get(int i)
// /
func (c *CommonTokenStream) Sync(i int) bool {
	var n = i - len(c.tokens) + 1 // how many more elements we need?
	if n > 0 {
		var fetched = c.fetch(n)
		if PortDebug {
			fmt.Println("Sync done")
		}
		return fetched >= n
	}
	return true
}

// Add {@code n} elements to buffer.
//
// @return The actual number of elements added to the buffer.
// /
func (c *CommonTokenStream) fetch(n int) int {
	if c.fetchedEOF {
		return 0
	}

	for i := 0; i < n; i++ {
		var t Token = c.tokenSource.NextToken()
		if PortDebug {
			fmt.Println("fetch loop")
		}
		t.SetTokenIndex(len(c.tokens))
		c.tokens = append(c.tokens, t)
		if t.GetTokenType() == TokenEOF {
			c.fetchedEOF = true
			return i + 1
		}
	}

	if PortDebug {
		fmt.Println("fetch done")
	}
	return n
}

// Get all tokens from start..stop inclusively///
func (c *CommonTokenStream) GetTokens(start int, stop int, types *IntervalSet) []Token {

	if start < 0 || stop < 0 {
		return nil
	}
	c.lazyInit()
	var subset = make([]Token, 0)
	if stop >= len(c.tokens) {
		stop = len(c.tokens) - 1
	}
	for i := start; i < stop; i++ {
		var t = c.tokens[i]
		if t.GetTokenType() == TokenEOF {
			break
		}
		if types == nil || types.contains(t.GetTokenType()) {
			subset = append(subset, t)
		}
	}
	return subset
}

func (c *CommonTokenStream) LA(i int) int {
	return c.LT(i).GetTokenType()
}

func (c *CommonTokenStream) lazyInit() {
	if c.index == -1 {
		c.setup()
	}
}

func (c *CommonTokenStream) setup() {
	c.Sync(0)
	c.index = c.adjustSeekIndex(0)
}

func (c *CommonTokenStream) GetTokenSource() TokenSource {
	return c.tokenSource
}

// Reset c token stream by setting its token source.///
func (c *CommonTokenStream) SetTokenSource(tokenSource TokenSource) {
	c.tokenSource = tokenSource
	c.tokens = make([]Token, 0)
	c.index = -1
}

// Given a starting index, return the index of the next token on channel.
// Return i if tokens[i] is on channel. Return -1 if there are no tokens
// on channel between i and EOF.
// /
func (c *CommonTokenStream) NextTokenOnChannel(i, channel int) int {
	c.Sync(i)
	if i >= len(c.tokens) {
		return -1
	}
	var token = c.tokens[i]
	for token.GetChannel() != c.channel {
		if token.GetTokenType() == TokenEOF {
			return -1
		}
		i += 1
		c.Sync(i)
		token = c.tokens[i]
	}
	return i
}

// Given a starting index, return the index of the previous token on channel.
// Return i if tokens[i] is on channel. Return -1 if there are no tokens
// on channel between i and 0.
func (c *CommonTokenStream) previousTokenOnChannel(i, channel int) int {
	for i >= 0 && c.tokens[i].GetChannel() != channel {
		i -= 1
	}
	return i
}

// Collect all tokens on specified channel to the right of
// the current token up until we see a token on DEFAULT_TOKEN_CHANNEL or
// EOF. If channel is -1, find any non default channel token.
func (c *CommonTokenStream) getHiddenTokensToRight(tokenIndex, channel int) []Token {
	c.lazyInit()
	if tokenIndex < 0 || tokenIndex >= len(c.tokens) {
		panic(strconv.Itoa(tokenIndex) + " not in 0.." + strconv.Itoa(len(c.tokens)-1))
	}
	var nextOnChannel = c.NextTokenOnChannel(tokenIndex+1, LexerDefaultTokenChannel)
	var from_ = tokenIndex + 1
	// if none onchannel to right, nextOnChannel=-1 so set to = last token
	var to int
	if nextOnChannel == -1 {
		to = len(c.tokens) - 1
	} else {
		to = nextOnChannel
	}
	return c.filterForChannel(from_, to, channel)
}

// Collect all tokens on specified channel to the left of
// the current token up until we see a token on DEFAULT_TOKEN_CHANNEL.
// If channel is -1, find any non default channel token.
func (c *CommonTokenStream) getHiddenTokensToLeft(tokenIndex, channel int) []Token {
	c.lazyInit()
	if tokenIndex < 0 || tokenIndex >= len(c.tokens) {
		panic(strconv.Itoa(tokenIndex) + " not in 0.." + strconv.Itoa(len(c.tokens)-1))
	}
	var prevOnChannel = c.previousTokenOnChannel(tokenIndex-1, LexerDefaultTokenChannel)
	if prevOnChannel == tokenIndex-1 {
		return nil
	}
	// if none on channel to left, prevOnChannel=-1 then from=0
	var from_ = prevOnChannel + 1
	var to = tokenIndex - 1
	return c.filterForChannel(from_, to, channel)
}

func (c *CommonTokenStream) filterForChannel(left, right, channel int) []Token {
	var hidden = make([]Token, 0)
	for i := left; i < right+1; i++ {
		var t = c.tokens[i]
		if channel == -1 {
			if t.GetChannel() != LexerDefaultTokenChannel {
				hidden = append(hidden, t)
			}
		} else if t.GetChannel() == channel {
			hidden = append(hidden, t)
		}
	}
	if len(hidden) == 0 {
		return nil
	}
	return hidden
}

func (c *CommonTokenStream) GetSourceName() string {
	return c.tokenSource.GetSourceName()
}

func (c *CommonTokenStream) Size() int {
	return len(c.tokens)
}

func (c *CommonTokenStream) Index() int {
	return c.index
}

func (c *CommonTokenStream) GetAllText() string {
	return c.GetTextFromInterval(nil)
}

func (c *CommonTokenStream) GetTextFromTokens(start, end Token) string {
	if start == nil || end == nil {
		return ""
	}

	return c.GetTextFromInterval(NewInterval(start.GetTokenIndex(), end.GetTokenIndex()))
}

func (c *CommonTokenStream) GetTextFromRuleContext(interval RuleContext) string {
	return c.GetTextFromInterval(interval.GetSourceInterval())
}

func (c *CommonTokenStream) GetTextFromInterval(interval *Interval) string {

	c.lazyInit()
	c.Fill()
	if interval == nil {
		interval = NewInterval(0, len(c.tokens)-1)
	}

	var start = interval.start
	var stop = interval.stop
	if start < 0 || stop < 0 {
		return ""
	}
	if stop >= len(c.tokens) {
		stop = len(c.tokens) - 1
	}

	var s = ""
	for i := start; i < stop+1; i++ {
		var t = c.tokens[i]
		if t.GetTokenType() == TokenEOF {
			break
		}
		s += t.GetText()
	}

	return s
}

// Get all tokens from lexer until EOF///
func (c *CommonTokenStream) Fill() {
	c.lazyInit()
	for c.fetch(1000) == 1000 {
		continue
	}
}

func (c *CommonTokenStream) adjustSeekIndex(i int) int {
	return c.NextTokenOnChannel(i, c.channel)
}

func (c *CommonTokenStream) LB(k int) Token {

	if k == 0 || c.index-k < 0 {
		return nil
	}
	var i = c.index
	var n = 1
	// find k good tokens looking backwards
	for n <= k {
		// Skip off-channel tokens
		i = c.previousTokenOnChannel(i-1, c.channel)
		n += 1
	}
	if i < 0 {
		return nil
	}
	return c.tokens[i]
}

func (c *CommonTokenStream) LT(k int) Token {
	c.lazyInit()
	if k == 0 {
		return nil
	}
	if k < 0 {
		return c.LB(-k)
	}
	var i = c.index
	var n = 1 // we know tokens[pos] is a good one
	// find k good tokens
	for n < k {
		// Skip off-channel tokens, but make sure to not look past EOF
		if c.Sync(i + 1) {
			i = c.NextTokenOnChannel(i+1, c.channel)
		}
		n += 1
	}
	return c.tokens[i]
}

// Count EOF just once.///
func (c *CommonTokenStream) getNumberOfOnChannelTokens() int {
	var n = 0
	c.Fill()
	for i := 0; i < len(c.tokens); i++ {
		var t = c.tokens[i]
		if t.GetChannel() == c.channel {
			n += 1
		}
		if t.GetTokenType() == TokenEOF {
			break
		}
	}
	return n
}
