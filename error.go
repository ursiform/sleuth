// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

import "fmt"

const (
	// Warnings are in the 800-899 range.
	warnInterface = 801
	warnClose     = 802
	// Errors are in the 900-999 range.
	errNew              = 900
	errCreate           = 901
	errDispatch         = 902
	errService          = 903
	errInitialize       = 904
	errStart            = 905
	errJoin             = 906
	errInterface        = 907
	errPort             = 908
	errNodeHeader       = 909
	errServiceHeader    = 910
	errVersionHeader    = 911
	errGroupHeader      = 912
	errVerbose          = 913
	errDispatchHeader   = 914
	errDispatchAction   = 915
	errScheme           = 916
	errResUnmarshal     = 917
	errResUnmarshalJSON = 918
	errUnknownService   = 919
	errTimeout          = 920
	errRECV             = 921
	errREPL             = 922
	errLogLevel         = 923
	errAdd              = 924
	errReqMarshal       = 925
	errReqUnmarshal     = 926
	errReqUnmarshalJSON = 927
	errReqUnmarshalHTTP = 928
	errReqWhisper       = 929
	errResWhisper       = 930
	errLeave            = 931
	errUnzip            = 932
	errUnzipRead        = 933
)

// Error is the type all sleuth errors can be asserted as in order to query
// the error code trace that resulted in any particular error. For example, a
// call to the New function may fail and its resultant error can then be
// checked:
// 	err := sleuth.New(nil)
// 	if err != nil {
//		fmt.Printf("%v\n", err.(*sleuth.Error).Codes)
//		// Outputs an integer slice.
// 	}
type Error struct {
	// Codes contains the list of error codes that led to a specific error.
	Codes   []int
	message string
}

// Error returns an error string.
func (e *Error) Error() string {
	return fmt.Sprintf("sleuth: %s %v", e.message, e.Codes)
}

func (e *Error) escalate(code int) *Error {
	e.Codes = append(e.Codes, code)
	return e
}

func newError(code int, format string, v ...interface{}) *Error {
	err := &Error{Codes: make([]int, 1), message: fmt.Sprintf(format, v...)}
	err.Codes[0] = code
	return err
}
