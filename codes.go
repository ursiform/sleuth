// Copyright 2016 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package sleuth

const (
	// Warnings are in the 800-899 range.
	warnInterface = 801
	warnClose     = 802
	warnAdd       = 803
	// Errors are in the 900-999 range.
	errServiceUndefined  = 900
	errInitialize        = 901
	errStart             = 902
	errJoin              = 903
	errInterface         = 904
	errPort              = 905
	errNodeHeader        = 906
	errServiceHeader     = 907
	errVersionHeader     = 908
	errGroupHeader       = 909
	errVerbose           = 910
	errREPL              = 911
	errDispatchHeader    = 912
	errDispatchAction    = 913
	errUnsupportedScheme = 914
	errUnknownService    = 915
	errTimeout           = 916
	errRECVUnmarshal     = 917
	errRECVHandle        = 918
	errLogLevel          = 919
)
