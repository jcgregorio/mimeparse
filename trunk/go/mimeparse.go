//
// This module provides basic functions for handling mime-types. It can handle
// matching mime-types against a list of media-ranges. See section 14.1 of
// the HTTP specification [RFC 2616] for a complete explanation.
//
//    http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.1
//
// Contents:
//     - ParseMimeType():     Parses a mime-type into its component parts.
//     - ParseMediaRange():   Media-ranges are mime-types with wild-cards and a 'q' quality parameter.
//     - Quality():           Determines the quality ('q') of a mime-type when compared against a list of media-ranges.
//     - QualityParsed():     Just like quality() except the second parameter must be pre-parsed.
//     - BestMatch():         Choose the mime-type with the highest quality ('q') from a list of candidates.

package mimeparse

import (
	"os"
	"strings"
	"strconv"
)

func ht(list []string) (head string, tail []string) {
	if len(list) < 1 {
		return "", []string{}
	} else if len(list) == 1 {
		return list[0], []string{}
	}
	return list[0], list[1:len(list)]
}

type Mime struct {
	// major type
	mtype string
	// subtype
	subtype string
	// parameters
	params map[string]string
}

// Carves up a mime-type and returns a struct of the
// (type, subtype, params) where 'params' is a dictionary
// of all the parameters for the media range.
// For example, the media range 'application/xhtml;q=0.5' would
// get parsed into:
//
// Mime {'application', 'xhtml', {'q', '0.5'}}, nil
func ParseMimeType(mimetype string) (parsed Mime, err os.Error) {
	full_type, parts := ht(strings.Split(mimetype, ";", -1))
	full_type = strings.ToLower(full_type)
	params := make(map[string]string)
	for _, s := range parts {
		subparts := strings.Split(s, "=", 2)
		if len(subparts) == 2 {
			params[strings.ToLower(strings.TrimSpace(subparts[0]))] = strings.TrimSpace(subparts[1])
		} else {
			params[strings.ToLower(strings.TrimSpace(subparts[0]))] = ""
		}
	}
	if strings.TrimSpace(full_type) == "*" {
		full_type = "*/*"
	}
	list := strings.Split(full_type, "/", -1)
	if len(list) != 2 {
		return Mime{"", "", map[string]string{"q": "0"}}, os.NewError("Not a valid mimetype")
	}
	maintype, subtype := list[0], list[1]
	return Mime{strings.TrimSpace(maintype), strings.TrimSpace(subtype), params}, nil
}

// Carves up a media range and returns a tuple of the
// (type, subtype, params) where 'params' is a dictionary
// of all the parameters for the media range.
// For example, the media range 'application/*;q=0.5' would
// get parsed into:
//
// ('application', '*', {'q', '0.5'})
//
// In addition this function also guarantees that there
// is a value for 'q' in the params dictionary, filling it
// in with a proper default if necessary.
func ParseMediaRange(mediarange string) (mime Mime, err os.Error) {
	parsed, err := ParseMimeType(mediarange)
	if err != nil {
		return parsed, err
	}
	if q, ok := parsed.params["q"]; ok {
		if val, err := strconv.Atof(q); err != nil || val > 1.0 || val < 0.0 {
			parsed.params["q"] = "1"
		}
	} else {
		parsed.params["q"] = "1"
	}
	return parsed, nil
}


// Find the best match for a given mime-type against
// a list of media_ranges that have already been
// parsed by ParseMediaRange(). Returns a tuple of
// the fitness value and the value of the 'q' quality
// parameter of the best match, or (-1, 0) if no match
// was found. Just as for QualityParsed(), 'parsedranges'
// must be a list of parsed media ranges.
func FitnessAndQuality(mimetype string, parsedRanges []Mime) (fitness int, quality float) {
	bestfitness := -1
	bestquality := 0.0
	target, _ := ParseMediaRange(mimetype)
	for _, r := range parsedRanges {
		pmatches := 0
		fitness := 0
		if (r.mtype == target.mtype || r.mtype == "*" || target.mtype == "*") &&
			(r.subtype == target.subtype || r.subtype == "*" || target.subtype == "*") {
			fitness += 1
			for key, targetvalue := range target.params {
				if key != "q" {
					if value, ok := r.params[key]; ok && value == targetvalue {
						pmatches++
					}
				}
			}
			fitness += pmatches
			if r.subtype == target.subtype {
				fitness += 10
			}
			if r.mtype == target.mtype {
				fitness += 100
			}
			if fitness > bestfitness {
				bestfitness = fitness
				bestquality, _ = strconv.Atof(r.params["q"])
			}
		}
	}

	return bestfitness, bestquality
}

//    Find the best match for a given mime-type against
//    a list of media_ranges that have already been
//    parsed by ParseMediaRange(). Returns the
//    'q' quality parameter of the best match, 0 if no
//    match was found. This function bahaves the same as quality()
//    except that 'parsed_ranges' must be a list of
//    parsed media ranges.
func QualityParsed(mimetype string, parsedRanges []Mime) (quality float) {
	_, quality = FitnessAndQuality(mimetype, parsedRanges)
	return
}

func ParseHeader(header string) (parsed []Mime) {
	ranges := strings.Split(header, ",", -1)
	parsed = make([]Mime, len(ranges))
	for i, r := range ranges {
		parsed[i], _ = ParseMediaRange(r)
	}
	return
}

// Returns the quality 'q' of a mime-type when compared
// against the media-ranges in ranges. For example:
//
// Quality('text/html','text/*;q=0.3, text/html;q=0.7, text/html;level=1, text/html;level=2;q=0.4, * / *;q=0.5')
// 0.7
func Quality(mimetype string, ranges string) (quality float) {
	return QualityParsed(mimetype, ParseHeader(ranges))
}

//  Takes a list of supported mime-types and finds the best
//  match for all the media-ranges listed in header. The value of
//  header must be a string that conforms to the format of the
//  HTTP Accept: header. The value of 'supported' is a list of
//  mime-types.
//
//  BestMatch(['application/xbel+xml', 'text/xml'], 'text/*;q=0.5,* /*; q=0.1')
//  'text/xml'
func BestMatch(supported []string, header string) string {
	parsedHeader := ParseHeader(header)
	if len(supported) == 0 {
		return ""
	}
	bestquality := 0.0
	bestmime := ""
	for _, mime := range supported {
		_, quality := FitnessAndQuality(mime, parsedHeader)
		if quality > bestquality {
			bestquality = quality
			bestmime = mime
		}
	}

	return bestmime
}
