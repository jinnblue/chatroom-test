package acascii

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jinnblue/chatroom-test/pkg/pathmap"
)

var tests = []struct {
	name       string // 用于匹配源库的测试名(github.com/cloudflare/ahocorasick)
	dict       []string
	input      string
	matches    []string
	wantFilter string
	wantHas    bool
}{
	{
		"TestNoPatterns",
		[]string{},
		"",
		nil,
		"",
		false,
	},
	{
		"TestNoData",
		[]string{"foo", "baz", "bar"},
		"",
		nil,
		"",
		false,
	},
	{
		"TestSuffixes",
		[]string{"Superman", "uperman", "perman", "erman"},
		"The Man Of Steel: Superman",
		[]string{"Superman", "uperman", "perman", "erman"},
		"The Man Of Steel: ********",
		true,
	},
	{
		"TestPrefixes",
		[]string{"Superman", "Superma", "Superm", "Super"},
		"The Man Of Steel: Superman",
		[]string{"Super", "Superm", "Superma", "Superman"},
		"The Man Of Steel: ********",
		true,
	},
	{
		"TestInterior",
		[]string{"Steel", "tee", "e"},
		"The Man Of Steel: Superman",
		[]string{"e", "tee", "Steel"},
		"Th* Man Of *****: Sup*rman",
		true,
	},
	{
		"TestMatchAtStart",
		[]string{"The", "Th", "he"},
		"The Man Of Steel: Superman",
		[]string{"Th", "The", "he"},
		"*** Man Of Steel: Superman",
		true,
	},
	{
		"TestMatchAtEnd",
		[]string{"teel", "eel", "el"},
		"The Man Of Steel",
		[]string{"teel", "eel", "el"},
		"The Man Of S****",
		true,
	},
	{
		"TestOverlappingPatterns",
		[]string{"Man ", "n Of", "Of S"},
		"The Man Of Steel",
		[]string{"Man ", "n Of", "Of S"},
		"The ********teel",
		true,
	},
	{
		"TestMultipleMatches",
		[]string{"The", "Man", "an"},
		"A Man A Plan A Canal: Panama, which Man Planned The Canal",
		[]string{"Man", "an", "The"},
		"A *** A Pl** A C**al: P**ama, which *** Pl**ned *** C**al",
		true,
	},
	{
		"TestSingleCharacterMatches",
		[]string{"a", "M", "z"},
		"A Man A Plan A Canal: Panama, which Man Planned The Canal",
		[]string{"M", "a"},
		"* **n * Pl*n * C*n*l: P*n***, which **n Pl*nned The C*n*l",
		true,
	},
	{
		"TestNothingMatches",
		[]string{"baz", "bar", "foo"},
		"A Man A Plan A Canal: Panama, which Man Planned The Canal",
		nil,
		"A Man A Plan A Canal: Panama, which Man Planned The Canal",
		false,
	},
	{
		"TestSymbolSkip",
		[]string{"she", "her", "he", "fuck"},
		" F*u!c~k~! all uhe and usher",
		[]string{"she", "her", "he", "fuck"},
		" *******~! all u** and u****",
		true,
	},
	{
		"Wikipedia1",
		[]string{"a", "ab", "bc", "bca", "c", "caa"},
		"abccab",
		[]string{"a", "ab", "bc", "c"},
		"******",
		true,
	},
	{
		"Wikipedia2",
		[]string{"a", "ab", "bc", "bca", "c", "caa"},
		"bccab",
		[]string{"bc", "c", "a", "ab"},
		"*****",
		true,
	},
	{
		"Wikipedia3",
		[]string{"a", "ab", "bc", "bca", "c", "caa"},
		"bccb",
		[]string{"bc", "c"},
		"***b",
		true,
	},
	{
		"Browser1",
		[]string{"Mozilla", "Mac", "Macintosh", "Safari", "Sausage"},
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Safari/537.36",
		[]string{"Mozilla", "Mac", "Macintosh", "Safari"},
		"*******/5.0 (*********; Intel *** OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 ******/537.36",
		true,
	},
	{
		"Browser2",
		[]string{"Mozilla", "Mac", "Macintosh", "Safari", "Sausage"},
		"Mozilla/5.0 (Mac; Intel Mac OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Safari/537.36",
		[]string{"Mozilla", "Mac", "Safari"},
		"*******/5.0 (***; Intel *** OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 ******/537.36",
		true,
	},
	{
		"Browser3",
		[]string{"Mozilla", "Mac", "Macintosh", "Safari", "Sausage"},
		"Mozilla/5.0 (Moc; Intel Computer OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Safari/537.36",
		[]string{"Mozilla", "Safari"},
		"*******/5.0 (Moc; Intel Computer OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 ******/537.36",
		true,
	},
	{
		"Browser4",
		[]string{"Mozilla", "Mac", "Macintosh", "Safari", "Sausage"},
		"Mozilla/5.0 (Moc; Intel Computer OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Sofari/537.36",
		[]string{"Mozilla"},
		"*******/5.0 (Moc; Intel Computer OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Sofari/537.36",
		true,
	},
	{
		"Browser5",
		[]string{"Mozilla", "Mac", "Macintosh", "Safari", "Sausage"},
		"Mazilla/5.0 (Moc; Intel Computer OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Sofari/537.36",
		nil,
		"Mazilla/5.0 (Moc; Intel Computer OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Sofari/537.36",
		false,
	},
	{
		// 确保回溯有效并能匹配到"per".因"Superwoman"与"Superman"有部分匹配,一些实现会有回溯匹配失败的问题
		"Backtrack",
		[]string{"Superwoman", "per"},
		"The Man Of Steel: Superman",
		[]string{"per"},
		"The Man Of Steel: Su***man",
		true,
	},
	{
		"NotAsciiInput",
		[]string{"Mozilla", "Mac", "Macintosh", "Safari", "Sausage", "Gecko"},
		"Mazilla/5.0 \u0000 (Moc; Intel Computer OS X 10_7_5) AppleWebKit/537.36 \uFFFF (KHTML, like Gecko) Chrome/30.0.1599.101 Sofari/537.36",
		[]string{"Gecko"},
		"Mazilla/5.0 \u0000 (Moc; Intel Computer OS X 10_7_5) AppleWebKit/537.36 \uFFFF (KHTML, like *****) Chrome/30.0.1599.101 Sofari/537.36",
		true,
	},
}

func TestAC(t *testing.T) {
	for _, tt := range tests {
		trie, err := NewACTrieFromStrings(tt.dict)
		if err != nil {
			t.Error(err)
		}

		contains := trie.HasBlackWord(tt.input)
		if !reflect.DeepEqual(contains, tt.wantHas) {
			t.Errorf("%s: HasBlackWord want %v, but got %v", tt.name, tt.wantHas, contains)
		}

		clean := trie.Filter(tt.input)
		if !reflect.DeepEqual(clean, tt.wantFilter) {
			t.Errorf(`%s: Filter want "%s", but got "%s"`, tt.name, tt.wantFilter, clean)
		}
	}
}

func TestNonASCIIDictionary(t *testing.T) {
	dict := []string{"hello world", "こんにちは世界"}
	_, err := NewACTrieFromStrings(dict)
	if err == nil {
		t.Errorf("字典包含非 ASCII 字符时未报错")
	}
}

func TestNewACTrieFromFile(t *testing.T) {
	abPath := pathmap.GetCurrentAbPath()
	filePath := filepath.Join(abPath, "../../", "internal/data/list.txt")
	trie := NewACTrieFromFile(filePath)
	if trie == nil {
		t.Errorf("通过文件 %s 构建失败", filePath)
		return
	}
	fmt.Println("dict file path:", filePath)

	tests := []struct {
		name       string // 用于匹配源库的测试名(github.com/cloudflare/ahocorasick)
		input      string
		wantFilter string
		wantHas    bool
	}{
		{
			"FileTestNoData",
			"",
			"",
			false,
		},
		{
			"FileTestBacktrack",
			"arsshitis,F#U$C$K!!!",
			"ars****is,*******!!!",
			true,
		},
	}

	for _, tt := range tests {
		contains := trie.HasBlackWord(tt.input)
		if !reflect.DeepEqual(contains, tt.wantHas) {
			t.Errorf("%s: HasBlackWord want %v, but got %v", tt.name, tt.wantHas, contains)
		}

		clean := trie.Filter(tt.input)
		if !reflect.DeepEqual(clean, tt.wantFilter) {
			t.Errorf(`%s: Filter want "%s", but got "%s"`, tt.name, tt.wantFilter, clean)
		}
	}
}

var (
	inStr    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Safari/537.36"
	inBytes  = Words(inStr)
	dict1    = []string{"Mozilla", "Mac", "Macintosh", "Safari", "Sausage"}
	dict2    = []string{"Googlebot", "bingbot", "msnbot", "Yandex", "Baiduspider"}
	trie1, _ = NewACTrieFromStrings(dict1)
	trie2, _ = NewACTrieFromStrings(dict2)
)

// 结果赋值，防止被编译器优化
var retHas bool

func BenchmarkAC1(b *testing.B) {
	var has bool
	for i := 0; i < b.N; i++ {
		has = trie1.HasBlackWord(inStr)
	}
	retHas = has
}

func BenchmarkAC2(b *testing.B) {
	var has bool
	for i := 0; i < b.N; i++ {
		has = trie2.HasBlackWord(inStr)
	}
	retHas = has
}
func BenchmarkAC2Byte(b *testing.B) {
	var has bool
	for i := 0; i < b.N; i++ {
		has = trie2.BytesHasBlackWord(inBytes)
	}
	retHas = has
}

var retClean string

func BenchmarkFilter1(b *testing.B) {
	var clean string
	for i := 0; i < b.N; i++ {
		clean = trie1.Filter(inStr)
	}
	retClean = clean
	// fmt.Println("BenchmarkFilter1:", retClean)
}

func BenchmarkFilter2(b *testing.B) {
	var clean string
	for i := 0; i < b.N; i++ {
		clean = trie2.Filter(inStr)
	}
	retClean = clean
	// fmt.Println("BenchmarkFilter2:", retClean)
}

func ExampleACTrie_HasBlackWord() {
	trie, _ := NewACTrieFromStrings([]string{"Superman", "uperman", "perman", "erman"})
	has := trie.HasBlackWord("The Man Of Steel: Superman")
	fmt.Println(has)
	// Output: true
}

func ExampleACTrie_BytesHasBlackWord() {
	trie, _ := NewACTrieFromStrings([]string{"Superman", "uperman", "perman", "erman"})
	has := trie.BytesHasBlackWord(Words("The Man Of Steel: Superman"))
	fmt.Println(has)
	// Output: true
}

func ExampleACTrie_Filter() {
	trie, _ := NewACTrieFromStrings([]string{"Superman", "uperman", "perman", "erman"})
	clean := trie.Filter("The Man Of Steel: Superman")
	fmt.Println(clean)
	// Output: The Man Of Steel: ********
}
