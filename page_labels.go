package oxpdf

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// PageLabels returns the label for each page, defaulting to one-based numbers.
func (d *Document) PageLabels() ([]string, error) {
	if d == nil {
		return nil, nil
	}
	labels := defaultPageLabels(d.NumPages())
	catalog, ok := d.Catalog()
	if !ok || catalog.PageLabels == nil {
		return labels, nil
	}
	objects := parseIndirectObjects(d.input)
	nums, err := collectPageLabelNums(objects, *catalog.PageLabels, map[objectRef]bool{})
	if err != nil {
		return nil, err
	}
	if len(nums) == 0 {
		return labels, nil
	}
	for i := range labels {
		spec, startIndex := pageLabelSpecForIndex(nums, i)
		if spec == nil {
			continue
		}
		labels[i] = spec.labelFor(i - startIndex)
	}
	return labels, nil
}

func collectPageLabelNums(objects []indirectObject, ref ObjectReference, seen map[objectRef]bool) ([]pageLabelNum, error) {
	key := objectRef{number: ref.Number, gen: ref.Generation}
	if seen[key] {
		return nil, unsupported(fmt.Sprintf("page label number tree cycle at %d %d R", ref.Number, ref.Generation))
	}
	seen[key] = true
	object, ok := indirectObjectByNumber(objects, ref.Number, ref.Generation)
	if !ok {
		return nil, unsupported(fmt.Sprintf("page labels node %d %d R is missing", ref.Number, ref.Generation))
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return nil, unsupported(fmt.Sprintf("page labels node %d is not a dictionary", object.number))
	}
	nums, err := parsePageLabelNums(dict)
	if err != nil {
		return nil, err
	}
	for _, kid := range directReferenceArray(dict, "Kids") {
		kidNums, err := collectPageLabelNums(objects, kid, seen)
		if err != nil {
			return nil, err
		}
		nums = append(nums, kidNums...)
	}
	sort.SliceStable(nums, func(i, j int) bool {
		return nums[i].index < nums[j].index
	})
	return nums, nil
}

type pageLabelNum struct {
	index int
	spec  pageLabelSpec
}

type pageLabelSpec struct {
	style  string
	prefix string
	start  int
}

func parsePageLabelNums(dict []byte) ([]pageLabelNum, error) {
	raw, ok := directArrayValue(dict, "Nums")
	if !ok {
		return nil, nil
	}
	i := 0
	nums := make([]pageLabelNum, 0)
	for {
		i = skipPDFSpace(raw, i)
		if i >= len(raw) {
			break
		}
		indexStart := i
		for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
			i++
		}
		if indexStart == i {
			return nil, unsupported("page label number tree has unsupported index")
		}
		index, err := strconv.Atoi(string(raw[indexStart:i]))
		if err != nil {
			return nil, unsupported("page label number tree has invalid index")
		}
		i = skipPDFSpace(raw, i)
		if i >= len(raw) || raw[i] != '<' || i+1 >= len(raw) || raw[i+1] != '<' {
			return nil, unsupported(fmt.Sprintf("page label index %d has unsupported value", index))
		}
		end, ok := scanDictionaryEnd(raw, i)
		if !ok {
			return nil, unsupported(fmt.Sprintf("page label index %d has malformed dictionary", index))
		}
		spec := parsePageLabelSpec(raw[i:end])
		nums = append(nums, pageLabelNum{index: index, spec: spec})
		i = end
	}
	sort.SliceStable(nums, func(i, j int) bool {
		return nums[i].index < nums[j].index
	})
	return nums, nil
}

func parsePageLabelSpec(dict []byte) pageLabelSpec {
	spec := pageLabelSpec{start: 1}
	if raw, ok := directNameValue(dict, "S"); ok {
		raw = bytes.TrimSpace(raw)
		if len(raw) > 1 && raw[0] == '/' {
			end := 1
			for end < len(raw) && !isPDFSpaceByte(raw[end]) && !isPDFDelimiterByte(raw[end]) {
				end++
			}
			spec.style = string(raw[1:end])
		}
	}
	if raw, ok := directNameValue(dict, "P"); ok {
		if prefix, ok := parsePDFTextValue(raw); ok {
			spec.prefix = prefix
		}
	}
	if start := directInteger(dict, "St"); start != nil && *start > 0 {
		spec.start = *start
	}
	return spec
}

func pageLabelSpecForIndex(nums []pageLabelNum, index int) (*pageLabelSpec, int) {
	var selected *pageLabelSpec
	startIndex := 0
	for i := range nums {
		if nums[i].index > index {
			break
		}
		selected = &nums[i].spec
		startIndex = nums[i].index
	}
	return selected, startIndex
}

func (s pageLabelSpec) labelFor(offset int) string {
	number := s.start + offset
	switch s.style {
	case "r":
		return s.prefix + strings.ToLower(romanNumeral(number))
	case "R":
		return s.prefix + romanNumeral(number)
	case "a":
		return s.prefix + strings.ToLower(alphaNumeral(number))
	case "A":
		return s.prefix + alphaNumeral(number)
	case "D", "":
		return s.prefix + strconv.Itoa(number)
	default:
		return s.prefix + strconv.Itoa(number)
	}
}

func defaultPageLabels(pageCount int) []string {
	labels := make([]string, pageCount)
	for i := range labels {
		labels[i] = strconv.Itoa(i + 1)
	}
	return labels
}

func romanNumeral(number int) string {
	if number <= 0 {
		return strconv.Itoa(number)
	}
	values := []struct {
		value int
		text  string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}
	var out strings.Builder
	for _, value := range values {
		for number >= value.value {
			out.WriteString(value.text)
			number -= value.value
		}
	}
	return out.String()
}

func alphaNumeral(number int) string {
	if number <= 0 {
		return strconv.Itoa(number)
	}
	var out []byte
	for number > 0 {
		number--
		out = append([]byte{byte('A' + number%26)}, out...)
		number /= 26
	}
	return string(out)
}
