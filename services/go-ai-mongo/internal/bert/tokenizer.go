// Package bert implements WordPiece tokenization and ONNX-based ESI prediction
// for Bio_ClinicalBERT (emilyalsentzer/Bio_ClinicalBERT, 5-class BertForSequenceClassification).
package bert

import (
	"bufio"
	"math"
	"os"
	"strings"
	"unicode"
)

const (
	clsToken = int64(101)
	sepToken = int64(102)
	padToken = int64(0)
	unkToken = int64(100)
	maxSeq   = 512
)

// LoadVocab reads vocab.txt (one token per line, 0-indexed) into a lookup map.
func LoadVocab(path string) (map[string]int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vocab := make(map[string]int64, 30000)
	var id int64
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)
	for scanner.Scan() {
		vocab[scanner.Text()] = id
		id++
	}
	return vocab, scanner.Err()
}

// Tokenize converts text to BERT input tensors of length maxSeq (512).
// Mirrors BertTokenizer(do_lower_case=True, tokenize_chinese_chars=True).
func Tokenize(vocab map[string]int64, text string) (inputIDs, attnMask, typeIDs []int64) {
	// Step 1: clean + lowercase + CJK spacing
	text = strings.ToLower(cleanText(addCJKSpaces(text)))

	// Step 2: whitespace-tokenize into words, then split each word on punctuation
	var wordTokens []string
	for _, word := range strings.Fields(text) {
		wordTokens = append(wordTokens, splitOnPunct(word)...)
	}

	// Step 3: WordPiece; reserve 2 slots for [CLS] and [SEP]
	tokens := []int64{clsToken}
	for _, w := range wordTokens {
		wp := wordPiece(vocab, w)
		if len(tokens)+len(wp)+1 > maxSeq {
			break
		}
		tokens = append(tokens, wp...)
	}
	tokens = append(tokens, sepToken)

	// Step 4: build fixed-length tensors (pad remainder)
	inputIDs = make([]int64, maxSeq)
	attnMask = make([]int64, maxSeq)
	typeIDs = make([]int64, maxSeq) // all zeros (single segment)
	for i, id := range tokens {
		inputIDs[i] = id
		attnMask[i] = 1
	}
	return
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func addCJKSpaces(text string) string {
	var b strings.Builder
	b.Grow(len(text) * 2)
	for _, r := range text {
		if isCJK(r) {
			b.WriteRune(' ')
			b.WriteRune(r)
			b.WriteRune(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cleanText(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		// Drop null bytes and non-printable control characters.
		if r == 0 || isControl(r) {
			continue
		}
		if unicode.IsSpace(r) {
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isControl(r rune) bool {
	if r == '\t' || r == '\n' || r == '\r' {
		return false
	}
	return unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r)
}

func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF) ||
		(r >= 0x2A700 && r <= 0x2B73F) ||
		(r >= 0xF900 && r <= 0xFAFF)
}

// splitOnPunct replicates BERT's _run_split_on_punc: split word at each
// punctuation character, keeping the punctuation as its own token.
func splitOnPunct(word string) []string {
	runes := []rune(word)
	var out []string
	var cur []rune
	for _, r := range runes {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			if len(cur) > 0 {
				out = append(out, string(cur))
				cur = cur[:0]
			}
			out = append(out, string(r))
		} else {
			cur = append(cur, r)
		}
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	if len(out) == 0 {
		return []string{word}
	}
	return out
}

// wordPiece runs the greedy longest-match WordPiece algorithm.
// If any subword is absent from vocab the whole token maps to [UNK].
func wordPiece(vocab map[string]int64, word string) []int64 {
	if id, ok := vocab[word]; ok {
		return []int64{id}
	}
	runes := []rune(word)
	var result []int64
	start := 0
	for start < len(runes) {
		end := len(runes)
		found := false
		for end > start {
			sub := string(runes[start:end])
			if start > 0 {
				sub = "##" + sub
			}
			if id, ok := vocab[sub]; ok {
				result = append(result, id)
				start = end
				found = true
				break
			}
			end--
		}
		if !found {
			return []int64{unkToken}
		}
	}
	return result
}

// Softmax converts raw logits to probabilities.
func Softmax(logits []float32) []float32 {
	max := logits[0]
	for _, v := range logits[1:] {
		if v > max {
			max = v
		}
	}
	var sum float64
	out := make([]float32, len(logits))
	for i, v := range logits {
		e := math.Exp(float64(v) - float64(max))
		out[i] = float32(e)
		sum += e
	}
	for i := range out {
		out[i] /= float32(sum)
	}
	return out
}
