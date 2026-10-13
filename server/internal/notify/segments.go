package notify

import (
	"strings"
	"unicode/utf16"
)

// SMS segment sizes. A message that fits in one segment gets the full payload;
// once it is split, each part gives up room to a header that reassembles it.
const (
	gsmSingleSegment = 160
	gsmMultiSegment  = 153
	ucsSingleSegment = 70
	ucsMultiSegment  = 67
)

// gsmBasic is the GSM 03.38 basic character set. Each costs one septet.
const gsmBasic = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?" +
	"¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"

// gsmExtension characters are sent as an escape plus a character, so each
// costs two septets.
const gsmExtension = "^{}\\[~]|€\f"

// SMSSegments reports how many segments a carrier bills for the message.
//
// Twilio charges per segment rather than per message, so this is what turns
// a count of alerts into a cost. A single character outside the GSM set, such
// as an emoji or a curly quote, switches the whole message to UCS-2 and more
// than halves the room in each segment.
func SMSSegments(message string) int32 {
	septets := 0

	for _, r := range message {
		switch {
		case strings.ContainsRune(gsmBasic, r):
			septets++
		case strings.ContainsRune(gsmExtension, r):
			septets += 2
		default:
			return segmentCount(len(utf16.Encode([]rune(message))), ucsSingleSegment, ucsMultiSegment)
		}
	}

	return segmentCount(septets, gsmSingleSegment, gsmMultiSegment)
}

func segmentCount(units, single, multi int) int32 {
	if units <= single {
		return 1
	}

	return int32((units + multi - 1) / multi) //nolint:gosec // bounded by message length
}
