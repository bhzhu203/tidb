// Copyright 2013 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package expression

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/context"
	"github.com/pingcap/tidb/mysql"
	"github.com/pingcap/tidb/sessionctx/variable"
	"github.com/pingcap/tidb/terror"
	"github.com/pingcap/tidb/util/types"
)

const ( // GET_FORMAT first argument.
	dateFormat      = "DATE"
	datetimeFormat  = "DATETIME"
	timestampFormat = "TIMESTAMP"
	timeFormat      = "TIME"
)

const ( // GET_FORMAT location.
	usaLocation      = "USA"
	jisLocation      = "JIS"
	isoLocation      = "ISO"
	eurLocation      = "EUR"
	internalLocation = "INTERNAL"
)

// DurationPattern determine whether to match the format of duration.
var DurationPattern = regexp.MustCompile(`^(|[-]?)(|\d{1,2}\s)(\d{2,3}:\d{2}:\d{2}|\d{1,2}:\d{2}|\d{1,6})(|\.\d*)$`)

var (
	_ functionClass = &dateFunctionClass{}
	_ functionClass = &dateDiffFunctionClass{}
	_ functionClass = &timeDiffFunctionClass{}
	_ functionClass = &dateFormatFunctionClass{}
	_ functionClass = &hourFunctionClass{}
	_ functionClass = &minuteFunctionClass{}
	_ functionClass = &secondFunctionClass{}
	_ functionClass = &microSecondFunctionClass{}
	_ functionClass = &monthFunctionClass{}
	_ functionClass = &monthNameFunctionClass{}
	_ functionClass = &nowFunctionClass{}
	_ functionClass = &dayNameFunctionClass{}
	_ functionClass = &dayOfMonthFunctionClass{}
	_ functionClass = &dayOfWeekFunctionClass{}
	_ functionClass = &dayOfYearFunctionClass{}
	_ functionClass = &weekFunctionClass{}
	_ functionClass = &weekDayFunctionClass{}
	_ functionClass = &weekOfYearFunctionClass{}
	_ functionClass = &yearFunctionClass{}
	_ functionClass = &yearWeekFunctionClass{}
	_ functionClass = &fromUnixTimeFunctionClass{}
	_ functionClass = &getFormatFunctionClass{}
	_ functionClass = &strToDateFunctionClass{}
	_ functionClass = &sysDateFunctionClass{}
	_ functionClass = &currentDateFunctionClass{}
	_ functionClass = &currentTimeFunctionClass{}
	_ functionClass = &timeFunctionClass{}
	_ functionClass = &utcDateFunctionClass{}
	_ functionClass = &utcTimestampFunctionClass{}
	_ functionClass = &extractFunctionClass{}
	_ functionClass = &arithmeticFunctionClass{}
	_ functionClass = &unixTimestampFunctionClass{}
	_ functionClass = &addTimeFunctionClass{}
	_ functionClass = &convertTzFunctionClass{}
	_ functionClass = &makeDateFunctionClass{}
	_ functionClass = &makeTimeFunctionClass{}
	_ functionClass = &periodAddFunctionClass{}
	_ functionClass = &periodDiffFunctionClass{}
	_ functionClass = &quarterFunctionClass{}
	_ functionClass = &secToTimeFunctionClass{}
	_ functionClass = &subTimeFunctionClass{}
	_ functionClass = &timeFormatFunctionClass{}
	_ functionClass = &timeToSecFunctionClass{}
	_ functionClass = &timestampAddFunctionClass{}
	_ functionClass = &toDaysFunctionClass{}
	_ functionClass = &toSecondsFunctionClass{}
	_ functionClass = &utcTimeFunctionClass{}
	_ functionClass = &timestampFunctionClass{}
)

var (
	_ builtinFunc = &builtinDateSig{}
	_ builtinFunc = &builtinDateDiffSig{}
	_ builtinFunc = &builtinTimeDiffSig{}
	_ builtinFunc = &builtinDateFormatSig{}
	_ builtinFunc = &builtinHourSig{}
	_ builtinFunc = &builtinMinuteSig{}
	_ builtinFunc = &builtinSecondSig{}
	_ builtinFunc = &builtinMicroSecondSig{}
	_ builtinFunc = &builtinMonthSig{}
	_ builtinFunc = &builtinMonthNameSig{}
	_ builtinFunc = &builtinNowWithArgSig{}
	_ builtinFunc = &builtinNowWithoutArgSig{}
	_ builtinFunc = &builtinDayNameSig{}
	_ builtinFunc = &builtinDayOfMonthSig{}
	_ builtinFunc = &builtinDayOfWeekSig{}
	_ builtinFunc = &builtinDayOfYearSig{}
	_ builtinFunc = &builtinWeekWithModeSig{}
	_ builtinFunc = &builtinWeekWithoutModeSig{}
	_ builtinFunc = &builtinWeekDaySig{}
	_ builtinFunc = &builtinWeekOfYearSig{}
	_ builtinFunc = &builtinYearSig{}
	_ builtinFunc = &builtinYearWeekWithModeSig{}
	_ builtinFunc = &builtinYearWeekWithoutModeSig{}
	_ builtinFunc = &builtinFromUnixTimeSig{}
	_ builtinFunc = &builtinGetFormatSig{}
	_ builtinFunc = &builtinStrToDateSig{}
	_ builtinFunc = &builtinSysDateWithFspSig{}
	_ builtinFunc = &builtinSysDateWithoutFspSig{}
	_ builtinFunc = &builtinCurrentDateSig{}
	_ builtinFunc = &builtinCurrentTimeSig{}
	_ builtinFunc = &builtinTimeSig{}
	_ builtinFunc = &builtinUTCDateSig{}
	_ builtinFunc = &builtinUTCTimestampWithArgSig{}
	_ builtinFunc = &builtinUTCTimestampWithoutArgSig{}
	_ builtinFunc = &builtinExtractSig{}
	_ builtinFunc = &builtinArithmeticSig{}
	_ builtinFunc = &builtinUnixTimestampSig{}
	_ builtinFunc = &builtinAddTimeSig{}
	_ builtinFunc = &builtinConvertTzSig{}
	_ builtinFunc = &builtinMakeDateSig{}
	_ builtinFunc = &builtinMakeTimeSig{}
	_ builtinFunc = &builtinPeriodAddSig{}
	_ builtinFunc = &builtinPeriodDiffSig{}
	_ builtinFunc = &builtinQuarterSig{}
	_ builtinFunc = &builtinSecToTimeSig{}
	_ builtinFunc = &builtinSubTimeSig{}
	_ builtinFunc = &builtinTimeToSecSig{}
	_ builtinFunc = &builtinTimestampAddSig{}
	_ builtinFunc = &builtinToDaysSig{}
	_ builtinFunc = &builtinToSecondsSig{}
	_ builtinFunc = &builtinUTCTimeSig{}
	_ builtinFunc = &builtinTimestampSig{}
)

// handleInvalidTimeError reports error or warning depend on the context.
func handleInvalidTimeError(ctx context.Context, err error) error {
	if err == nil || !terror.ErrorEqual(err, types.ErrInvalidTimeFormat) {
		return err
	}
	sc := ctx.GetSessionVars().StmtCtx
	if ctx.GetSessionVars().StrictSQLMode && (sc.InInsertStmt || sc.InUpdateOrDeleteStmt) {
		return err
	}
	sc.AppendWarning(err)
	return nil
}

func convertTimeToMysqlTime(t time.Time, fsp int) (types.Time, error) {
	tr, err := types.RoundFrac(t, int(fsp))
	if err != nil {
		return types.Time{}, errors.Trace(err)
	}

	return types.Time{
		Time: types.FromGoTime(tr),
		Type: mysql.TypeDatetime,
		Fsp:  fsp,
	}, nil
}

func convertToTimeWithFsp(sc *variable.StatementContext, arg types.Datum, tp byte, fsp int) (d types.Datum, err error) {
	if fsp > types.MaxFsp {
		fsp = types.MaxFsp
	}

	f := types.NewFieldType(tp)
	f.Decimal = fsp

	d, err = arg.ConvertTo(sc, f)
	if err != nil {
		d.SetNull()
		return d, errors.Trace(err)
	}

	if d.IsNull() {
		return
	}

	if d.Kind() != types.KindMysqlTime {
		d.SetNull()
		return d, errors.Errorf("need time type, but got %T", d.GetValue())
	}
	return
}

func convertToTime(sc *variable.StatementContext, arg types.Datum, tp byte) (d types.Datum, err error) {
	return convertToTimeWithFsp(sc, arg, tp, types.MaxFsp)
}

func convertToDuration(sc *variable.StatementContext, arg types.Datum, fsp int) (d types.Datum, err error) {
	f := types.NewFieldType(mysql.TypeDuration)
	f.Decimal = fsp

	d, err = arg.ConvertTo(sc, f)
	if err != nil {
		d.SetNull()
		return d, errors.Trace(err)
	}

	if d.IsNull() {
		return
	}

	if d.Kind() != types.KindMysqlDuration {
		d.SetNull()
		return d, errors.Errorf("need duration type, but got %T", d.GetValue())
	}
	return
}

type dateFunctionClass struct {
	baseFunctionClass
}

func (c *dateFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinDateSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinDateSig struct {
	baseBuiltinFunc
}

// eval evals a builtinDateSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date
func (b *builtinDateSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	return convertToTime(b.ctx.GetSessionVars().StmtCtx, args[0], mysql.TypeDate)
}

func convertDatumToTime(sc *variable.StatementContext, d types.Datum) (t types.Time, err error) {
	if d.Kind() != types.KindMysqlTime {
		d, err = convertToTime(sc, d, mysql.TypeDatetime)
		if err != nil {
			return t, errors.Trace(err)
		}
	}
	return d.GetMysqlTime(), nil
}

type dateDiffFunctionClass struct {
	baseFunctionClass
}

func (c *dateDiffFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinDateDiffSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinDateDiffSig struct {
	baseBuiltinFunc
}

// eval evals a builtinDateDiffSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_datediff
func (b *builtinDateDiffSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	if args[0].IsNull() || args[1].IsNull() {
		return
	}
	sc := b.ctx.GetSessionVars().StmtCtx
	t1, err := convertDatumToTime(sc, args[0])
	if err != nil {
		return d, errors.Trace(err)
	}

	t2, err := convertDatumToTime(sc, args[1])
	if err != nil {
		return d, errors.Trace(err)
	}

	if t1.Time.Month() == 0 || t1.Time.Day() == 0 || t2.Time.Month() == 0 || t2.Time.Day() == 0 {
		return
	}

	r := types.DateDiff(t1.Time, t2.Time)
	d.SetInt64(int64(r))
	return
}

type timeDiffFunctionClass struct {
	baseFunctionClass
}

func (c *timeDiffFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinTimeDiffSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinTimeDiffSig struct {
	baseBuiltinFunc
}

func (b *builtinTimeDiffSig) getStrFsp(strArg string, fsp int) int {
	if n := strings.IndexByte(strArg, '.'); n >= 0 {
		lenStrFsp := len(strArg[n+1:])
		if lenStrFsp <= types.MaxFsp {
			fsp = int(math.Max(float64(lenStrFsp), float64(fsp)))
		}
	}
	return fsp
}

func (b *builtinTimeDiffSig) convertArgToTime(sc *variable.StatementContext, arg types.Datum, fsp int) (t types.Time, err error) {
	// Fix issue #3923, see https://github.com/pingcap/tidb/issues/3923,
	// TIMEDIFF() returns expr1 − expr2 expressed as a Duration value. expr1 and expr2 are Duration or date-and-time expressions,
	// but both must be of the same type. if expr is a string, we first try to convert it to Duration, if it failed,
	// we then try to convert it to Datetime
	switch arg.Kind() {
	case types.KindString, types.KindBytes:
		strArg := arg.GetString()
		fsp = b.getStrFsp(strArg, fsp)
		t, err = types.StrToDuration(sc, strArg, fsp)
	case types.KindMysqlDuration:
		t, err = arg.GetMysqlDuration().ConvertToTime(mysql.TypeDuration)
	default:
		t, err = convertDatumToTime(sc, arg)
	}
	return t, errors.Trace(err)
}

// eval evals a builtinTimeDiffSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_timediff
func (b *builtinTimeDiffSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	if args[0].IsNull() || args[1].IsNull() {
		return
	}

	sc := b.ctx.GetSessionVars().StmtCtx
	fsp := int(math.Max(float64(args[0].Frac()), float64(args[1].Frac())))
	t0, err := b.convertArgToTime(sc, args[0], fsp)
	if err != nil {
		return d, errors.Trace(err)
	}
	t1, err := b.convertArgToTime(sc, args[1], fsp)
	if err != nil {
		return d, errors.Trace(err)
	}
	if (types.IsTemporalWithDate(t0.Type) &&
		t1.Type == mysql.TypeDuration) ||
		(types.IsTemporalWithDate(t1.Type) &&
			t0.Type == mysql.TypeDuration) {
		return d, nil // Incompatible types, return NULL
	}

	t := t0.Sub(&t1)
	ret, truncated := types.TruncateOverflowMySQLTime(t.Duration)
	if truncated {
		err = types.ErrTruncatedWrongVal.GenByArgs("time", t.String())
		err = sc.HandleTruncate(err)
	}
	t.Duration = ret
	d.SetMysqlDuration(t)
	return
}

type dateFormatFunctionClass struct {
	baseFunctionClass
}

func (c *dateFormatFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinDateFormatSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinDateFormatSig struct {
	baseBuiltinFunc
}

// eval evals a builtinDateFormatSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date-format
func (b *builtinDateFormatSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	return builtinDateFormat(args, b.ctx)
}

// builtinDateFormat ...
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date-format
func builtinDateFormat(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	date, err := convertToTime(ctx.GetSessionVars().StmtCtx, args[0], mysql.TypeDatetime)
	if err != nil {
		return d, errors.Trace(err)
	}

	if date.IsNull() {
		return
	}
	t := date.GetMysqlTime()
	str, err := t.DateFormat(args[1].GetString())
	if err != nil {
		return d, errors.Trace(err)
	}
	d.SetString(str)
	return
}

type fromDaysFunctionClass struct {
	baseFunctionClass
}

func (c *fromDaysFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinFromDaysSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinFromDaysSig struct {
	baseBuiltinFunc
}

// eval evals a builtinFromDaysSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_from-days
func (b *builtinFromDaysSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	days, err := args[0].ToInt64(b.ctx.GetSessionVars().StmtCtx)
	d.SetMysqlTime(types.TimeFromDays(days))
	return d, nil
}

type hourFunctionClass struct {
	baseFunctionClass
}

func (c *hourFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpDuration)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 3, 0
	sig := &builtinHourSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinHourSig struct {
	baseIntBuiltinFunc
}

// evalInt evals HOUR(time).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_hour
func (b *builtinHourSig) evalInt(row []types.Datum) (int64, bool, error) {
	dur, isNull, err := b.args[0].EvalDuration(row, b.ctx.GetSessionVars().StmtCtx)
	// ignore error and return NULL
	if isNull || err != nil {
		return 0, true, nil
	}
	return int64(dur.Hour()), false, nil
}

type minuteFunctionClass struct {
	baseFunctionClass
}

func (c *minuteFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpDuration)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 2, 0
	sig := &builtinMinuteSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinMinuteSig struct {
	baseIntBuiltinFunc
}

// evalInt evals MINUTE(time).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_minute
func (b *builtinMinuteSig) evalInt(row []types.Datum) (int64, bool, error) {
	dur, isNull, err := b.args[0].EvalDuration(row, b.ctx.GetSessionVars().StmtCtx)
	// ignore error and return NULL
	if isNull || err != nil {
		return 0, true, nil
	}
	return int64(dur.Minute()), false, nil
}

type secondFunctionClass struct {
	baseFunctionClass
}

func (c *secondFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpDuration)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 2, 0
	sig := &builtinSecondSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinSecondSig struct {
	baseIntBuiltinFunc
}

// evalInt evals SECOND(time).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_second
func (b *builtinSecondSig) evalInt(row []types.Datum) (int64, bool, error) {
	dur, isNull, err := b.args[0].EvalDuration(row, b.ctx.GetSessionVars().StmtCtx)
	// ignore error and return NULL
	if isNull || err != nil {
		return 0, true, nil
	}
	return int64(dur.Second()), false, nil
}

type microSecondFunctionClass struct {
	baseFunctionClass
}

func (c *microSecondFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpDuration)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 6, 0
	sig := &builtinMicroSecondSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinMicroSecondSig struct {
	baseIntBuiltinFunc
}

// evalInt evals MICROSECOND(expr).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_microsecond
func (b *builtinMicroSecondSig) evalInt(row []types.Datum) (int64, bool, error) {
	dur, isNull, err := b.args[0].EvalDuration(row, b.ctx.GetSessionVars().StmtCtx)
	// ignore error and return NULL
	if isNull || err != nil {
		return 0, true, nil
	}
	return int64(dur.MicroSecond()), false, nil
}

type monthFunctionClass struct {
	baseFunctionClass
}

func (c *monthFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 2, 0
	sig := &builtinMonthSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinMonthSig struct {
	baseIntBuiltinFunc
}

// evalInt evals MONTH(date).
// see: https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_month
func (b *builtinMonthSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.getCtx().GetSessionVars().StmtCtx
	date, isNull, err := b.args[0].EvalTime(row, sc)

	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}

	if date.IsZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}

	return int64(date.Time.Month()), false, nil
}

// builtinMonth ...
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_month
func builtinMonth(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	d, err = convertToTime(ctx.GetSessionVars().StmtCtx, args[0], mysql.TypeDate)
	if err != nil || d.IsNull() {
		return d, errors.Trace(err)
	}

	// No need to check type here.
	t := d.GetMysqlTime()
	i := int64(0)
	if t.IsZero() {
		d.SetInt64(i)
		return
	}
	i = int64(t.Time.Month())
	d.SetInt64(i)
	return
}

type monthNameFunctionClass struct {
	baseFunctionClass
}

func (c *monthNameFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinMonthNameSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinMonthNameSig struct {
	baseBuiltinFunc
}

func (b *builtinMonthNameSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	return builtinMonthName(args, b.ctx)
}

// builtinMonthName ...
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_monthname
func builtinMonthName(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	d, err = builtinMonth(args, ctx)
	if err != nil || d.IsNull() {
		return d, errors.Trace(err)
	}

	mon := int(d.GetInt64())
	if mon <= 0 || mon > len(types.MonthNames) {
		d.SetNull()
		if mon == 0 {
			return
		}
		return d, errors.Errorf("no name for invalid month: %d.", mon)
	}
	d.SetString(types.MonthNames[mon-1])

	return
}

type dayNameFunctionClass struct {
	baseFunctionClass
}

func (c *dayNameFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinDayNameSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinDayNameSig struct {
	baseBuiltinFunc
}

// eval evals a builtinDayNameSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_dayname
func (b *builtinDayNameSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	d, err = builtinWeekDay(args, b.ctx)
	if err != nil || d.IsNull() {
		return d, errors.Trace(err)
	}
	weekday := d.GetInt64()
	if (weekday < 0) || (weekday >= int64(len(types.WeekdayNames))) {
		d.SetNull()
		return d, errors.Errorf("no name for invalid weekday: %d.", weekday)
	}
	d.SetString(types.WeekdayNames[weekday])
	return
}

type dayOfMonthFunctionClass struct {
	baseFunctionClass
}

func (c *dayOfMonthFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen = 2
	sig := &builtinDayOfMonthSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinDayOfMonthSig struct {
	baseIntBuiltinFunc
}

// evalInt evals a builtinDayOfMonthSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_dayofmonth
func (b *builtinDayOfMonthSig) evalInt(row []types.Datum) (int64, bool, error) {
	arg, isNull, err := b.args[0].EvalTime(row, b.ctx.GetSessionVars().StmtCtx)
	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	if arg.IsZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}
	return int64(arg.Time.Day()), false, nil
}

type dayOfWeekFunctionClass struct {
	baseFunctionClass
}

func (c *dayOfWeekFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen = 1
	sig := &builtinDayOfWeekSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinDayOfWeekSig struct {
	baseIntBuiltinFunc
}

// evalInt evals a builtinDayOfWeekSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_dayofweek
func (b *builtinDayOfWeekSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.ctx.GetSessionVars().StmtCtx
	arg, isNull, err := b.args[0].EvalTime(row, sc)
	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	if arg.InvalidZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}
	// 1 is Sunday, 2 is Monday, .... 7 is Saturday
	return int64(arg.Time.Weekday() + 1), false, nil
}

type dayOfYearFunctionClass struct {
	baseFunctionClass
}

func (c *dayOfYearFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen = 3
	sig := &builtinDayOfYearSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinDayOfYearSig struct {
	baseIntBuiltinFunc
}

// evalInt evals a builtinDayOfYearSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_dayofyear
func (b *builtinDayOfYearSig) evalInt(row []types.Datum) (int64, bool, error) {
	arg, isNull, err := b.args[0].EvalTime(row, b.ctx.GetSessionVars().StmtCtx)
	if isNull || err != nil {
		return 0, isNull, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	if arg.InvalidZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}

	return int64(arg.Time.YearDay()), false, nil
}

type weekFunctionClass struct {
	baseFunctionClass
}

func (c *weekFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}

	argTps := []evalTp{tpTime}
	if len(args) == 2 {
		argTps = append(argTps, tpInt)
	}

	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, argTps...)
	if err != nil {
		return nil, errors.Trace(err)
	}

	bf.tp.Flen, bf.tp.Decimal = 2, 0

	var sig builtinFunc
	if len(args) == 2 {
		sig = &builtinWeekWithModeSig{baseIntBuiltinFunc{bf}}
	} else {
		sig = &builtinWeekWithoutModeSig{baseIntBuiltinFunc{bf}}
	}
	return sig.setSelf(sig), nil
}

type builtinWeekWithModeSig struct {
	baseIntBuiltinFunc
}

// evalInt evals WEEK(date, mode).
// see: https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_week
func (b *builtinWeekWithModeSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.ctx.GetSessionVars().StmtCtx
	date, isNull, err := b.args[0].EvalTime(row, sc)

	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}

	if date.IsZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}

	mode, isNull, err := b.args[1].EvalInt(row, sc)
	if isNull || err != nil {
		return 0, isNull, errors.Trace(err)
	}

	week := date.Time.Week(int(mode))
	return int64(week), false, nil
}

type builtinWeekWithoutModeSig struct {
	baseIntBuiltinFunc
}

// evalInt evals WEEK(date).
// see: https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_week
func (b *builtinWeekWithoutModeSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.ctx.GetSessionVars().StmtCtx
	date, isNull, err := b.args[0].EvalTime(row, sc)

	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}

	if date.IsZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}

	week := date.Time.Week(0)
	return int64(week), false, nil
}

type weekDayFunctionClass struct {
	baseFunctionClass
}

func (c *weekDayFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinWeekDaySig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinWeekDaySig struct {
	baseBuiltinFunc
}

func (b *builtinWeekDaySig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	return builtinWeekDay(args, b.ctx)
}

// builtinWeekDay ...
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_weekday
func builtinWeekDay(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	d, err = convertToTime(ctx.GetSessionVars().StmtCtx, args[0], mysql.TypeDate)
	if err != nil || d.IsNull() {
		return d, errors.Trace(err)
	}

	// No need to check type here.
	t := d.GetMysqlTime()
	if t.IsZero() {
		// TODO: log warning or return error?
		d.SetNull()
		return
	}

	// Monday is 0, ... Sunday = 6 in MySQL
	// but in go, Sunday is 0, ... Saturday is 6
	// w will do a conversion.
	w := (int64(t.Time.Weekday()) + 6) % 7
	d.SetInt64(w)
	return
}

type weekOfYearFunctionClass struct {
	baseFunctionClass
}

func (c *weekOfYearFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 2, 0
	sig := &builtinWeekOfYearSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinWeekOfYearSig struct {
	baseIntBuiltinFunc
}

// evalInt evals WEEKOFYEAR(date).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_weekofyear
func (b *builtinWeekOfYearSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.ctx.GetSessionVars().StmtCtx
	date, isNull, err := b.args[0].EvalTime(row, sc)

	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}

	if date.IsZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}

	week := date.Time.Week(3)
	return int64(week), false, nil
}

type yearFunctionClass struct {
	baseFunctionClass
}

func (c *yearFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 4, 0
	sig := &builtinYearSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinYearSig struct {
	baseIntBuiltinFunc
}

// evalInt evals YEAR(date).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_year
func (b *builtinYearSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.getCtx().GetSessionVars().StmtCtx
	date, isNull, err := b.args[0].EvalTime(row, sc)

	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}

	if date.IsZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}

	return int64(date.Time.Year()), false, nil
}

type yearWeekFunctionClass struct {
	baseFunctionClass
}

func (c *yearWeekFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := []evalTp{tpTime}
	if len(args) == 2 {
		argTps = append(argTps, tpInt)
	}

	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, argTps...)
	if err != nil {
		return nil, errors.Trace(err)
	}

	bf.tp.Flen, bf.tp.Decimal = 6, 0

	var sig builtinFunc
	if len(args) == 2 {
		sig = &builtinYearWeekWithModeSig{baseIntBuiltinFunc{bf}}
	} else {
		sig = &builtinYearWeekWithoutModeSig{baseIntBuiltinFunc{bf}}
	}
	return sig.setSelf(sig), nil
}

type builtinYearWeekWithModeSig struct {
	baseIntBuiltinFunc
}

// evalInt evals YEARWEEK(date,mode).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_yearweek
func (b *builtinYearWeekWithModeSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.ctx.GetSessionVars().StmtCtx

	date, isNull, err := b.args[0].EvalTime(row, sc)
	if isNull || err != nil {
		return 0, isNull, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	if date.IsZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}

	mode, isNull, err := b.args[1].EvalInt(row, sc)
	if err != nil {
		return 0, true, errors.Trace(err)
	}
	if isNull {
		mode = 0
	}

	year, week := date.Time.YearWeek(int(mode))
	result := int64(week + year*100)
	if result < 0 {
		return int64(math.MaxUint32), false, nil
	}
	return result, false, nil
}

type builtinYearWeekWithoutModeSig struct {
	baseIntBuiltinFunc
}

// evalInt evals YEARWEEK(date).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_yearweek
func (b *builtinYearWeekWithoutModeSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.ctx.GetSessionVars().StmtCtx

	date, isNull, err := b.args[0].EvalTime(row, sc)
	if err != nil {
		sc.AppendWarning(err)
	}
	if isNull || err != nil || date.InvalidZero() {
		return 0, true, nil
	}

	year, week := date.Time.YearWeek(0)
	result := int64(week + year*100)
	if result < 0 {
		return int64(math.MaxUint32), false, nil
	}
	return result, false, nil
}

type fromUnixTimeFunctionClass struct {
	baseFunctionClass
}

func (c *fromUnixTimeFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinFromUnixTimeSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinFromUnixTimeSig struct {
	baseBuiltinFunc
}

// eval evals a builtinFromUnixTimeSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_from-unixtime
func (b *builtinFromUnixTimeSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	sc := b.ctx.GetSessionVars().StmtCtx
	unixTimeStamp, err := args[0].ToDecimal(sc)
	if err != nil {
		return d, errors.Trace(err)
	}
	// 0 <= unixTimeStamp <= INT32_MAX
	if unixTimeStamp.IsNegative() {
		return
	}
	integralPart, err := unixTimeStamp.ToInt()
	if err == types.ErrTruncated {
		err = nil
	}
	if err != nil {
		return d, errors.Trace(err)
	}
	if integralPart > int64(math.MaxInt32) {
		return
	}
	// Split the integral part and fractional part of a decimal timestamp.
	// e.g. for timestamp 12345.678,
	// first get the integral part 12345,
	// then (12345.678 - 12345) * (10^9) to get the decimal part and convert it to nanosecond precision.
	integerDecimalTp := new(types.MyDecimal).FromInt(integralPart)
	fracDecimalTp := new(types.MyDecimal)
	err = types.DecimalSub(unixTimeStamp, integerDecimalTp, fracDecimalTp)
	if err != nil {
		return d, errors.Trace(err)
	}
	nano := new(types.MyDecimal).FromInt(int64(time.Second))
	x := new(types.MyDecimal)
	err = types.DecimalMul(fracDecimalTp, nano, x)
	if err != nil {
		return d, errors.Trace(err)
	}
	fractionalPart, err := x.ToInt() // here fractionalPart is result multiplying the original fractional part by 10^9.
	if err == types.ErrTruncated {
		err = nil
	}
	if err != nil {
		return d, errors.Trace(err)
	}

	_, fracDigitsNumber := unixTimeStamp.PrecisionAndFrac()
	fsp := fracDigitsNumber
	if fracDigitsNumber > types.MaxFsp {
		fsp = types.MaxFsp
	}

	t, err := convertTimeToMysqlTime(time.Unix(integralPart, fractionalPart), fsp)
	if err != nil {
		return d, errors.Trace(err)
	}

	if args[0].Kind() == types.KindString { // Keep consistent with MySQL.
		t.Fsp = types.MaxFsp
	}
	d.SetMysqlTime(t)
	if len(args) == 1 {
		return
	}
	return builtinDateFormat([]types.Datum{d, args[1]}, b.ctx)
}

type getFormatFunctionClass struct {
	baseFunctionClass
}

func (c *getFormatFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinGetFormatSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinGetFormatSig struct {
	baseBuiltinFunc
}

// eval evals a builtinGetFormatSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_get-format
func (b *builtinGetFormatSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	t := args[0].GetString()
	l := args[1].GetString()
	switch t {
	case dateFormat:
		switch l {
		case usaLocation:
			d.SetString("%m.%d.%Y")
		case jisLocation:
			d.SetString("%Y-%m-%d")
		case isoLocation:
			d.SetString("%Y-%m-%d")
		case eurLocation:
			d.SetString("%d.%m.%Y")
		case internalLocation:
			d.SetString("%Y%m%d")
		}
	case datetimeFormat, timestampFormat:
		switch l {
		case usaLocation:
			d.SetString("%Y-%m-%d %H.%i.%s")
		case jisLocation:
			d.SetString("%Y-%m-%d %H:%i:%s")
		case isoLocation:
			d.SetString("%Y-%m-%d %H:%i:%s")
		case eurLocation:
			d.SetString("%Y-%m-%d %H.%i.%s")
		case internalLocation:
			d.SetString("%Y%m%d%H%i%s")
		}
	case timeFormat:
		switch l {
		case usaLocation:
			d.SetString("%h:%i:%s %p")
		case jisLocation:
			d.SetString("%H:%i:%s")
		case isoLocation:
			d.SetString("%H:%i:%s")
		case eurLocation:
			d.SetString("%H.%i.%s")
		case internalLocation:
			d.SetString("%H%i%s")
		}
	}

	return
}

type strToDateFunctionClass struct {
	baseFunctionClass
}

func (c *strToDateFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinStrToDateSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinStrToDateSig struct {
	baseBuiltinFunc
}

// eval evals a builtinStrToDateSig.
// See https://dev.mysql.com/doc/refman/5.5/en/date-and-time-functions.html#function_str-to-date
func (b *builtinStrToDateSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	date := args[0].GetString()
	format := args[1].GetString()
	var t types.Time

	succ := t.StrToDate(date, format)
	if !succ {
		d.SetNull()
		return
	}

	d.SetMysqlTime(t)
	return
}

type sysDateFunctionClass struct {
	baseFunctionClass
}

func (c *sysDateFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := []evalTp{}
	if len(args) == 1 {
		argTps = append(argTps, tpInt)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpTime, argTps...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 19, 0
	bf.deterministic = false

	var sig builtinFunc
	if len(args) == 1 {
		sig = &builtinSysDateWithFspSig{baseTimeBuiltinFunc{bf}}
	} else {
		sig = &builtinSysDateWithoutFspSig{baseTimeBuiltinFunc{bf}}
	}
	return sig.setSelf(sig), nil
}

type builtinSysDateWithFspSig struct {
	baseTimeBuiltinFunc
}

// evalTime evals SYSDATE(fsp).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_sysdate
func (b *builtinSysDateWithFspSig) evalTime(row []types.Datum) (d types.Time, isNull bool, err error) {
	sc := b.ctx.GetSessionVars().StmtCtx

	fsp, isNull, err := b.args[0].EvalInt(row, sc)
	if isNull || err != nil {
		return types.Time{}, isNull, errors.Trace(err)
	}

	result, err := convertTimeToMysqlTime(time.Now(), int(fsp))
	if err != nil {
		return types.Time{}, true, errors.Trace(err)
	}
	return result, false, nil
}

type builtinSysDateWithoutFspSig struct {
	baseTimeBuiltinFunc
}

// evalTime evals SYSDATE().
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_sysdate
func (b *builtinSysDateWithoutFspSig) evalTime(row []types.Datum) (d types.Time, isNull bool, err error) {
	result, err := convertTimeToMysqlTime(time.Now(), 0)
	if err != nil {
		return types.Time{}, true, errors.Trace(err)
	}
	return result, false, nil
}

type currentDateFunctionClass struct {
	baseFunctionClass
}

func (c *currentDateFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 10, 0
	bf.deterministic = false
	sig := &builtinCurrentDateSig{baseTimeBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinCurrentDateSig struct {
	baseTimeBuiltinFunc
}

// eval evals CURDATE().
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_curdate
func (b *builtinCurrentDateSig) evalTime(row []types.Datum) (d types.Time, isNull bool, err error) {
	year, month, day := time.Now().Date()
	result := types.Time{
		Time: types.FromDate(year, int(month), day, 0, 0, 0, 0),
		Type: mysql.TypeDate,
		Fsp:  0}
	return result, false, nil
}

type currentTimeFunctionClass struct {
	baseFunctionClass
}

func (c *currentTimeFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinCurrentTimeSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinCurrentTimeSig struct {
	baseBuiltinFunc
}

// eval evals a builtinCurrentTimeSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_curtime
func (b *builtinCurrentTimeSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	fsp := 0
	sc := b.ctx.GetSessionVars().StmtCtx
	if len(args) == 1 && !args[0].IsNull() {
		if fsp, err = checkFsp(sc, args[0]); err != nil {
			d.SetNull()
			return d, errors.Trace(err)
		}
	}
	d.SetString(time.Now().Format("15:04:05.000000"))
	return convertToDuration(b.ctx.GetSessionVars().StmtCtx, d, fsp)
}

type timeFunctionClass struct {
	baseFunctionClass
}

func (c *timeFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpDuration, tpString)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinTimeSig{baseDurationBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinTimeSig struct {
	baseDurationBuiltinFunc
}

// evalDuration evals a builtinTimeSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_time.
func (b *builtinTimeSig) evalDuration(row []types.Datum) (res types.Duration, isNull bool, err error) {
	sc := b.getCtx().GetSessionVars().StmtCtx
	expr, isNull, err := b.args[0].EvalString(row, sc)
	if isNull || err != nil {
		return res, isNull, errors.Trace(err)
	}

	fsp := 0
	if idx := strings.Index(expr, "."); idx != -1 {
		fsp = len(expr) - idx - 1
	}

	if fsp, err = types.CheckFsp(fsp); err != nil {
		return res, isNull, errors.Trace(err)
	}

	res, err = types.ParseDuration(expr, fsp)
	if types.ErrTruncatedWrongVal.Equal(err) {
		err = sc.HandleTruncate(err)
	}
	return res, isNull, errors.Trace(err)
}

type utcDateFunctionClass struct {
	baseFunctionClass
}

func (c *utcDateFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bf.tp.Flen, bf.tp.Decimal = 10, 0
	bf.deterministic = false
	sig := &builtinUTCDateSig{baseTimeBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinUTCDateSig struct {
	baseTimeBuiltinFunc
}

// evalTime evals UTC_DATE, UTC_DATE().
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_utc-date
func (b *builtinUTCDateSig) evalTime(row []types.Datum) (types.Time, bool, error) {
	year, month, day := time.Now().UTC().Date()
	result := types.Time{
		Time: types.FromGoTime(time.Date(year, month, day, 0, 0, 0, 0, time.UTC)),
		Type: mysql.TypeDate,
		Fsp:  types.UnspecifiedFsp}
	return result, false, nil
}

type utcTimestampFunctionClass struct {
	baseFunctionClass
}

func getFlenAndDecimal4UTCTimestampAndNow(sc *variable.StatementContext, arg Expression) (flen, decimal int) {
	if constant, ok := arg.(*Constant); ok {
		fsp, isNull, err := constant.EvalInt(nil, sc)
		if isNull || err != nil || fsp > int64(types.MaxFsp) {
			decimal = types.MaxFsp
		} else if fsp < int64(types.MinFsp) {
			decimal = types.MinFsp
		} else {
			decimal = int(fsp)
		}
	}
	if decimal > 0 {
		flen = 19 + 1 + decimal
	} else {
		flen = 19
	}
	return flen, decimal
}

func (c *utcTimestampFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := make([]evalTp, 0, 1)
	if len(args) == 1 {
		argTps = append(argTps, tpInt)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpTime, argTps...)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(args) == 1 {
		bf.tp.Flen, bf.tp.Decimal = getFlenAndDecimal4UTCTimestampAndNow(bf.ctx.GetSessionVars().StmtCtx, args[0])
	} else {
		bf.tp.Flen, bf.tp.Decimal = 19, 0
	}
	bf.deterministic = false

	var sig builtinFunc
	if len(args) == 1 {
		sig = &builtinUTCTimestampWithArgSig{baseTimeBuiltinFunc{bf}}
	} else {
		sig = &builtinUTCTimestampWithoutArgSig{baseTimeBuiltinFunc{bf}}
	}
	return sig.setSelf(sig), nil
}

func evalUTCTimestampWithFsp(fsp int) (types.Time, bool, error) {
	result, err := convertTimeToMysqlTime(time.Now().UTC(), fsp)
	if err != nil {
		return types.Time{}, true, errors.Trace(err)
	}
	return result, false, nil
}

type builtinUTCTimestampWithArgSig struct {
	baseTimeBuiltinFunc
}

// evalTime evals UTC_TIMESTAMP(fsp).
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_utc-timestamp
func (b *builtinUTCTimestampWithArgSig) evalTime(row []types.Datum) (types.Time, bool, error) {
	num, isNull, err := b.args[0].EvalInt(row, b.ctx.GetSessionVars().StmtCtx)
	if err != nil {
		return types.Time{}, true, errors.Trace(err)
	}

	if !isNull && num > int64(types.MaxFsp) {
		return types.Time{}, true, errors.Errorf("Too-big precision %v specified for 'utc_timestamp'. Maximum is %v.", num, types.MaxFsp)
	}
	if !isNull && num < int64(types.MinFsp) {
		return types.Time{}, true, errors.Errorf("Invalid negative %d specified, must in [0, 6].", num)
	}

	result, isNull, err := evalUTCTimestampWithFsp(int(num))
	return result, isNull, errors.Trace(err)
}

type builtinUTCTimestampWithoutArgSig struct {
	baseTimeBuiltinFunc
}

// evalTime evals UTC_TIMESTAMP().
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_utc-timestamp
func (b *builtinUTCTimestampWithoutArgSig) evalTime(row []types.Datum) (types.Time, bool, error) {
	result, isNull, err := evalUTCTimestampWithFsp(0)
	return result, isNull, errors.Trace(err)
}

type nowFunctionClass struct {
	baseFunctionClass
}

func (c *nowFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	argTps := make([]evalTp, 0, 1)
	if len(args) == 1 {
		argTps = append(argTps, tpInt)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpTime, argTps...)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(args) == 1 {
		bf.tp.Flen, bf.tp.Decimal = getFlenAndDecimal4UTCTimestampAndNow(bf.ctx.GetSessionVars().StmtCtx, args[0])
	} else {
		bf.tp.Flen, bf.tp.Decimal = 19, 0
	}
	bf.deterministic = false

	var sig builtinFunc
	if len(args) == 1 {
		sig = &builtinNowWithArgSig{baseTimeBuiltinFunc{bf}}
	} else {
		sig = &builtinNowWithoutArgSig{baseTimeBuiltinFunc{bf}}
	}
	return sig.setSelf(sig), nil
}

func evalNowWithFsp(ctx context.Context, fsp int) (types.Time, bool, error) {
	sysTs, err := getSystemTimestamp(ctx)
	if err != nil {
		return types.Time{}, true, errors.Trace(err)
	}

	result, err := convertTimeToMysqlTime(sysTs, fsp)
	if err != nil {
		return types.Time{}, true, errors.Trace(err)
	}

	err = result.ConvertTimeZone(time.Local, ctx.GetSessionVars().GetTimeZone())
	if err != nil {
		return types.Time{}, true, errors.Trace(err)
	}

	return result, false, nil
}

type builtinNowWithArgSig struct {
	baseTimeBuiltinFunc
}

// evalTime evals NOW(fsp)
// see: https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_now
func (b *builtinNowWithArgSig) evalTime(row []types.Datum) (types.Time, bool, error) {
	fsp, isNull, err := b.args[0].EvalInt(row, b.ctx.GetSessionVars().StmtCtx)

	if err != nil {
		return types.Time{}, true, errors.Trace(err)
	}

	if isNull {
		fsp = 0
	} else if fsp > int64(types.MaxFsp) {
		return types.Time{}, true, errors.Errorf("Too-big precision %v specified for 'now'. Maximum is %v.", fsp, types.MaxFsp)
	} else if fsp < int64(types.MinFsp) {
		return types.Time{}, true, errors.Errorf("Invalid negative %d specified, must in [0, 6].", fsp)
	}

	result, isNull, err := evalNowWithFsp(b.ctx, int(fsp))
	return result, isNull, errors.Trace(err)
}

type builtinNowWithoutArgSig struct {
	baseTimeBuiltinFunc
}

// evalTime evals NOW()
// see: https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_now
func (b *builtinNowWithoutArgSig) evalTime(row []types.Datum) (types.Time, bool, error) {
	result, isNull, err := evalNowWithFsp(b.ctx, 0)
	return result, isNull, errors.Trace(err)
}

type extractFunctionClass struct {
	baseFunctionClass
}

func (c *extractFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinExtractSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinExtractSig struct {
	baseBuiltinFunc
}

// eval evals a builtinExtractSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_extract
func (b *builtinExtractSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	unit := args[0].GetString()
	vd := args[1]

	if vd.IsNull() {
		d.SetNull()
		return
	}

	f := types.NewFieldType(mysql.TypeDatetime)
	f.Decimal = types.MaxFsp
	val, err := vd.ConvertTo(b.ctx.GetSessionVars().StmtCtx, f)
	if err != nil {
		d.SetNull()
		return d, errors.Trace(err)
	}
	if val.IsNull() {
		d.SetNull()
		return
	}

	if val.Kind() != types.KindMysqlTime {
		d.SetNull()
		return d, errors.Errorf("need time type, but got %T", val)
	}
	t := val.GetMysqlTime()
	n, err1 := types.ExtractTimeNum(unit, t)
	if err1 != nil {
		d.SetNull()
		return d, errors.Trace(err1)
	}
	d.SetInt64(n)
	return
}

// TODO: duplicate with types.CheckFsp, better use types.CheckFsp.
func checkFsp(sc *variable.StatementContext, arg types.Datum) (int, error) {
	fsp, err := arg.ToInt64(sc)
	if err != nil {
		return 0, errors.Trace(err)
	}
	if int(fsp) > types.MaxFsp {
		return 0, errors.Errorf("Too big precision %d specified. Maximum is 6.", fsp)
	} else if fsp < 0 {
		return 0, errors.Errorf("Invalid negative %d specified, must in [0, 6].", fsp)
	}
	return int(fsp), nil
}

type dateArithFunctionClass struct {
	baseFunctionClass

	op ast.DateArithType
}

func (c *dateArithFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinDateArithSig{newBaseBuiltinFunc(args, ctx), c.op}
	return sig.setSelf(sig), nil
}

type builtinDateArithSig struct {
	baseBuiltinFunc

	op ast.DateArithType
}

func (b *builtinDateArithSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	// args[0] -> Date
	// args[1] -> Interval Value
	// args[2] -> Interval Unit
	// health check for date and interval
	if args[0].IsNull() || args[1].IsNull() {
		return
	}
	nodeDate := args[0]
	nodeIntervalValue := args[1]
	nodeIntervalUnit := args[2].GetString()
	if nodeIntervalValue.IsNull() {
		return
	}
	// parse date
	fieldType := mysql.TypeDate
	var resultField *types.FieldType
	switch nodeDate.Kind() {
	case types.KindMysqlTime:
		x := nodeDate.GetMysqlTime()
		if (x.Type == mysql.TypeDatetime) || (x.Type == mysql.TypeTimestamp) {
			fieldType = mysql.TypeDatetime
		}
	case types.KindString:
		x := nodeDate.GetString()
		if !types.IsDateFormat(x) {
			fieldType = mysql.TypeDatetime
		}
	case types.KindInt64:
		x := nodeDate.GetInt64()
		if t, err1 := types.ParseTimeFromInt64(x); err1 == nil {
			if (t.Type == mysql.TypeDatetime) || (t.Type == mysql.TypeTimestamp) {
				fieldType = mysql.TypeDatetime
			}
		}
	}
	sc := b.ctx.GetSessionVars().StmtCtx
	if types.IsClockUnit(nodeIntervalUnit) {
		fieldType = mysql.TypeDatetime
	}
	resultField = types.NewFieldType(fieldType)
	resultField.Decimal = types.MaxFsp
	value, err := nodeDate.ConvertTo(b.ctx.GetSessionVars().StmtCtx, resultField)
	if err != nil {
		return d, errInvalidOperation.Gen("DateArith invalid args, need date but get %T", nodeDate)
	}
	if value.IsNull() {
		return d, errInvalidOperation.Gen("DateArith invalid args, need date but get %v", value.GetValue())
	}
	if value.Kind() != types.KindMysqlTime {
		return d, errInvalidOperation.Gen("DateArith need time type, but got %T", value.GetValue())
	}
	result := value.GetMysqlTime()
	// parse interval
	var interval string
	if strings.ToLower(nodeIntervalUnit) == "day" {
		day, err1 := parseDayInterval(sc, nodeIntervalValue)
		if err1 != nil {
			return d, errInvalidOperation.Gen("DateArith invalid day interval, need int but got %T", nodeIntervalValue.GetString())
		}
		interval = fmt.Sprintf("%d", day)
	} else {
		if nodeIntervalValue.Kind() == types.KindString {
			interval = fmt.Sprintf("%v", nodeIntervalValue.GetString())
		} else {
			ii, err1 := nodeIntervalValue.ToInt64(sc)
			if err1 != nil {
				return d, errors.Trace(err1)
			}
			interval = fmt.Sprintf("%v", ii)
		}
	}
	year, month, day, dur, err := types.ExtractTimeValue(nodeIntervalUnit, interval)
	if err != nil {
		return d, errors.Trace(err)
	}
	if b.op == ast.DateArithSub {
		year, month, day, dur = -year, -month, -day, -dur
	}
	// TODO: Consider time_zone variable.
	t, err := result.Time.GoTime(time.Local)
	if err != nil {
		return d, errors.Trace(err)
	}
	t = t.Add(dur)
	t = t.AddDate(int(year), int(month), int(day))
	if t.Nanosecond() == 0 {
		result.Fsp = 0
	}
	result.Time = types.FromGoTime(t)
	d.SetMysqlTime(result)
	return
}

var reg = regexp.MustCompile(`[\d]+`)

func parseDayInterval(sc *variable.StatementContext, value types.Datum) (int64, error) {
	switch value.Kind() {
	case types.KindString:
		vs := value.GetString()
		s := strings.ToLower(vs)
		if s == "false" {
			return 0, nil
		} else if s == "true" {
			return 1, nil
		}
		value.SetString(reg.FindString(vs))
	}
	return value.ToInt64(sc)
}

type timestampDiffFunctionClass struct {
	baseFunctionClass
}

func (c *timestampDiffFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpString, tpTime, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinTimestampDiffSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinTimestampDiffSig struct {
	baseIntBuiltinFunc
}

// evalInt evals a builtinTimestampDiffSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_timestampdiff
func (b *builtinTimestampDiffSig) evalInt(row []types.Datum) (int64, bool, error) {
	ctx := b.getCtx().GetSessionVars().StmtCtx
	unit, isNull, err := b.args[0].EvalString(row, ctx)
	if isNull || err != nil {
		return 0, isNull, errors.Trace(err)
	}
	t1, isNull, err := b.args[1].EvalTime(row, ctx)
	if isNull || err != nil {
		return 0, isNull, errors.Trace(handleInvalidTimeError(b.getCtx(), err))
	}
	t2, isNull, err := b.args[2].EvalTime(row, ctx)
	if isNull || err != nil {
		return 0, isNull, errors.Trace(handleInvalidTimeError(b.getCtx(), err))
	}
	if t1.InvalidZero() || t2.InvalidZero() {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}
	return types.TimestampDiff(unit, t1, t2), false, nil
}

type unixTimestampFunctionClass struct {
	baseFunctionClass
}

func (c *unixTimestampFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinUnixTimestampSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinUnixTimestampSig struct {
	baseBuiltinFunc
}

// eval evals a builtinUnixTimestampSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_unix-timestamp
func (b *builtinUnixTimestampSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	if len(args) == 0 {
		now := time.Now().Unix()
		d.SetInt64(now)
		return
	}

	var (
		t  types.Time
		t1 time.Time
	)
	switch args[0].Kind() {
	case types.KindString:
		t, err = types.ParseTime(args[0].GetString(), mysql.TypeDatetime, types.MaxFsp)
		if err != nil {
			return d, errors.Trace(err)
		}
	case types.KindInt64, types.KindUint64:
		t, err = types.ParseTimeFromInt64(args[0].GetInt64())
		if err != nil {
			return d, errors.Trace(err)
		}
	case types.KindMysqlTime:
		t = args[0].GetMysqlTime()
	case types.KindNull:
		return
	default:
		return d, errors.Errorf("Unknown args type for unix_timestamp %d", args[0].Kind())
	}

	t1, err = t.Time.GoTime(getTimeZone(b.ctx))
	if err != nil {
		d.SetInt64(0)
		return d, nil
	}

	if t.Time.Microsecond() > 0 {
		var dec types.MyDecimal
		dec.FromFloat64(float64(t1.Unix()) + float64(t.Time.Microsecond())/1e6)
		d.SetMysqlDecimal(&dec)
	} else {
		d.SetInt64(t1.Unix())
	}
	return
}

type timestampFunctionClass struct {
	baseFunctionClass
}

func (c *timestampFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinTimestampSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinTimestampSig struct {
	baseBuiltinFunc
}

// eval evals a builtinTimestampSig.
// See https://dev.mysql.com/doc/refman/5.5/en/date-and-time-functions.html#function_timestamp
func (b *builtinTimestampSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	if args[0].IsNull() {
		return
	}
	var arg0 types.Time
	switch tp := args[0].Kind(); tp {
	case types.KindInt64, types.KindUint64:
		arg0, err = types.ParseDatetimeFromNum(args[0].GetInt64())
		if err != nil {
			return d, errors.Trace(err)
		}
	case types.KindString, types.KindBytes, types.KindMysqlDecimal, types.KindFloat32, types.KindFloat64:
		s, err1 := args[0].ToString()
		if err1 != nil {
			return d, errors.Trace(err1)
		}
		arg0, err = types.ParseTime(s, mysql.TypeDatetime, getFsp(s))
		if err != nil {
			return d, errors.Trace(err)
		}
	case types.KindMysqlTime:
		arg0 = args[0].GetMysqlTime()
	default:
		return d, errors.Errorf("Unknown args type for timestamp %d", tp)
	}
	if len(args) == 1 {
		d.SetMysqlTime(arg0)
		return
	}
	// If exists args[1].
	s, err := args[1].ToString()
	if err != nil {
		return d, errors.Trace(err)
	}
	arg1, err := types.ParseDuration(s, getFsp(s))
	if err != nil {
		return d, errors.Trace(err)
	}
	tmpDuration := arg0.Add(arg1)
	result, err := tmpDuration.ConvertToTime(arg0.Type)
	if err != nil {
		return d, errors.Trace(err)
	}
	d.SetMysqlTime(result)
	return
}

func getFsp(s string) (fsp int) {
	fsp = len(s) - strings.Index(s, ".") - 1
	if fsp == len(s) {
		fsp = 0
	} else if fsp > 6 {
		fsp = 6
	}
	return
}

func getTimeZone(ctx context.Context) *time.Location {
	ret := ctx.GetSessionVars().TimeZone
	if ret == nil {
		ret = time.Local
	}
	return ret
}

// isDuration returns a boolean indicating whether the str matches the format of duration.
// See https://dev.mysql.com/doc/refman/5.7/en/time.html
func isDuration(str string) bool {
	return DurationPattern.MatchString(str)
}

// strDatetimeAddDuration adds duration to datetime string, returns a datum value.
func strDatetimeAddDuration(d string, arg1 types.Duration) (result types.Datum, err error) {
	arg0, err := types.ParseTime(d, mysql.TypeDatetime, getFsp(d))
	if err != nil {
		return result, errors.Trace(err)
	}
	tmpDuration := arg0.Add(arg1)
	resultDuration, err := tmpDuration.ConvertToTime(mysql.TypeDatetime)
	if err != nil {
		return result, errors.Trace(err)
	}
	if getFsp(d) != 0 {
		tmpDuration.Fsp = types.MaxFsp
	} else {
		tmpDuration.Fsp = types.MinFsp
	}
	result.SetString(resultDuration.String())
	return
}

// strDurationAddDuration adds duration to duration string, returns a datum value.
func strDurationAddDuration(d string, arg1 types.Duration) (result types.Datum, err error) {
	arg0, err := types.ParseDuration(d, getFsp(d))
	if err != nil {
		return result, errors.Trace(err)
	}
	tmpDuration, err := arg0.Add(arg1)
	if err != nil {
		return result, errors.Trace(err)
	}
	if getFsp(d) != 0 {
		tmpDuration.Fsp = types.MaxFsp
	} else {
		tmpDuration.Fsp = types.MinFsp
	}
	result.SetString(tmpDuration.String())
	return
}

// strDatetimeSubDuration subtracts duration from datetime string, returns a datum value.
func strDatetimeSubDuration(d string, arg1 types.Duration) (result types.Datum, err error) {
	arg0, err := types.ParseTime(d, mysql.TypeDatetime, getFsp(d))
	if err != nil {
		return result, errors.Trace(err)
	}
	arg1time, err := arg1.ConvertToTime(uint8(getFsp(arg1.String())))
	if err != nil {
		return result, errors.Trace(err)
	}
	tmpDuration := arg0.Sub(&arg1time)
	resultDuration, err := tmpDuration.ConvertToTime(mysql.TypeDatetime)
	if err != nil {
		return result, errors.Trace(err)
	}
	if getFsp(d) != 0 {
		tmpDuration.Fsp = types.MaxFsp
	} else {
		tmpDuration.Fsp = types.MinFsp
	}
	result.SetString(resultDuration.String())
	return
}

// strDurationSubDuration subtracts duration from duration string, returns a datum value.
func strDurationSubDuration(d string, arg1 types.Duration) (result types.Datum, err error) {
	arg0, err := types.ParseDuration(d, getFsp(d))
	if err != nil {
		return result, errors.Trace(err)
	}
	tmpDuration, err := arg0.Sub(arg1)
	if err != nil {
		return result, errors.Trace(err)
	}
	if getFsp(d) != 0 {
		tmpDuration.Fsp = types.MaxFsp
	} else {
		tmpDuration.Fsp = types.MinFsp
	}
	result.SetString(tmpDuration.String())
	return
}

type addTimeFunctionClass struct {
	baseFunctionClass
}

func (c *addTimeFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinAddTimeSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinAddTimeSig struct {
	baseBuiltinFunc
}

// eval evals a builtinAddTimeSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_addtime
func (b *builtinAddTimeSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	if args[0].IsNull() || args[1].IsNull() {
		return
	}
	var arg1 types.Duration
	switch tp := args[1].Kind(); tp {
	case types.KindMysqlDuration:
		arg1 = args[1].GetMysqlDuration()
	default:
		s, err := args[1].ToString()
		if err != nil {
			return d, errors.Trace(err)
		}
		if getFsp(s) == 0 {
			arg1, err = types.ParseDuration(s, 0)
		} else {
			arg1, err = types.ParseDuration(s, types.MaxFsp)
		}
		if err != nil {
			return d, errors.Trace(err)
		}

	}
	switch tp := args[0].Kind(); tp {
	case types.KindMysqlTime:
		arg0 := args[0].GetMysqlTime()
		tmpDuration := arg0.Add(arg1)
		result, err := tmpDuration.ConvertToTime(arg0.Type)
		if err != nil {
			return d, errors.Trace(err)
		}
		d.SetMysqlTime(result)
	case types.KindMysqlDuration:
		arg0 := args[0].GetMysqlDuration()
		result, err := arg0.Add(arg1)
		if err != nil {
			return d, errors.Trace(err)
		}
		d.SetMysqlDuration(result)
	default:
		ss, err := args[0].ToString()
		if err != nil {
			return d, errors.Trace(err)
		}
		if isDuration(ss) {
			return strDurationAddDuration(ss, arg1)
		}
		return strDatetimeAddDuration(ss, arg1)
	}
	return
}

type convertTzFunctionClass struct {
	baseFunctionClass
}

func (c *convertTzFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinConvertTzSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinConvertTzSig struct {
	baseBuiltinFunc
}

// eval evals a builtinConvertTzSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_convert-tz
func (b *builtinConvertTzSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}

	if args[0].IsNull() || args[1].IsNull() || args[2].IsNull() {
		return
	}

	sc := b.ctx.GetSessionVars().StmtCtx

	fsp := 0
	if args[0].Kind() == types.KindString {
		fsp = types.DateFSP(args[0].GetString())
	}

	arg0, err := convertToTimeWithFsp(sc, args[0], mysql.TypeDatetime, fsp)
	if err != nil {
		return d, errors.Trace(err)
	}

	if arg0.IsNull() {
		return
	}

	dt := arg0.GetMysqlTime()

	fromTZ := args[1].GetString()
	toTZ := args[2].GetString()

	const tzArgReg = `(^(\+|-)(0?[0-9]|1[0-2]):[0-5]?\d$)|(^\+13:00$)`
	r, _ := regexp.Compile(tzArgReg)
	fmatch := r.MatchString(fromTZ)
	tmatch := r.MatchString(toTZ)

	if !fmatch && !tmatch {
		ftz, err := time.LoadLocation(fromTZ)
		if err != nil {
			return d, errors.Trace(err)
		}

		ttz, err := time.LoadLocation(toTZ)
		if err != nil {
			return d, errors.Trace(err)
		}

		t, err := dt.Time.GoTime(ftz)
		if err != nil {
			return d, errors.Trace(err)
		}

		d.SetMysqlTime(types.Time{
			Time: types.FromGoTime(t.In(ttz)),
			Type: mysql.TypeDatetime,
			Fsp:  dt.Fsp,
		})
		return d, nil
	}

	if fmatch && tmatch {
		t, err := dt.Time.GoTime(time.Local)
		if err != nil {
			return d, errors.Trace(err)
		}

		d.SetMysqlTime(types.Time{
			Time: types.FromGoTime(t.Add(timeZone2Duration(toTZ) - timeZone2Duration(fromTZ))),
			Type: mysql.TypeDatetime,
			Fsp:  dt.Fsp,
		})
	}

	return
}

type makeDateFunctionClass struct {
	baseFunctionClass
}

func (c *makeDateFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpTime, tpInt, tpInt)
	if err != nil {
		return nil, errors.Trace(err)
	}
	tp := bf.tp
	tp.Tp, tp.Flen, tp.Decimal = mysql.TypeDate, mysql.MaxDateWidth, 0
	sig := &builtinMakeDateSig{baseTimeBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinMakeDateSig struct {
	baseTimeBuiltinFunc
}

// evalTime evaluates a builtinMakeDateSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_makedate
func (b *builtinMakeDateSig) evalTime(row []types.Datum) (d types.Time, isNull bool, err error) {
	args := b.getArgs()
	sc := b.ctx.GetSessionVars().StmtCtx
	var year, dayOfYear int64
	year, isNull, err = args[0].EvalInt(row, sc)
	if isNull || err != nil {
		return d, true, errors.Trace(err)
	}
	dayOfYear, isNull, err = args[1].EvalInt(row, sc)
	if isNull || err != nil {
		return d, true, errors.Trace(err)
	}
	if dayOfYear <= 0 || year < 0 || year > 9999 {
		return d, true, nil
	}
	if year < 70 {
		year += 2000
	} else if year < 100 {
		year += 1900
	}
	startTime := types.Time{
		Time: types.FromDate(int(year), 1, 1, 0, 0, 0, 0),
		Type: mysql.TypeDate,
		Fsp:  0,
	}
	retTimestamp := types.TimestampDiff("DAY", types.ZeroDate, startTime)
	if retTimestamp == 0 {
		return d, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}
	ret := types.TimeFromDays(retTimestamp + dayOfYear - 1)
	if ret.IsZero() || ret.Time.Year() > 9999 {
		return d, true, nil
	}
	return ret, false, nil
}

type makeTimeFunctionClass struct {
	baseFunctionClass
}

func (c *makeTimeFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinMakeTimeSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinMakeTimeSig struct {
	baseBuiltinFunc
}

// eval evals a builtinMakeTimeSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_maketime
func (b *builtinMakeTimeSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	// MAKETIME(hour, minute, second)
	if args[0].IsNull() || args[1].IsNull() || args[2].IsNull() {
		return
	}

	var (
		hour     int64
		minute   int64
		second   float64
		overflow bool
	)
	sc := b.ctx.GetSessionVars().StmtCtx
	hour, _ = args[0].ToInt64(sc)
	// MySQL TIME datatype: https://dev.mysql.com/doc/refman/5.7/en/time.html
	// ranges from '-838:59:59.000000' to '838:59:59.000000'
	if hour < -838 {
		hour = -838
		overflow = true
	} else if hour > 838 {
		hour = 838
		overflow = true
	}

	minute, _ = args[1].ToInt64(sc)
	if minute < 0 || minute >= 60 {
		return
	}

	second, _ = args[2].ToFloat64(sc)
	if second < 0 || second >= 60 {
		return
	}
	if hour == -838 || hour == 838 {
		if second > 59 {
			second = 59
		}
	}
	if overflow {
		minute = 59
		second = 59
	}

	var dur types.Duration
	fsp := types.MaxFsp
	if args[2].Kind() != types.KindString {
		sec, _ := args[2].ToString()
		secs := strings.Split(sec, ".")
		if len(secs) <= 1 {
			fsp = 0
		} else if len(secs[1]) < fsp {
			fsp = len(secs[1])
		}
	}
	dur, err = types.ParseDuration(fmt.Sprintf("%02d:%02d:%v", hour, minute, second), fsp)
	if err != nil {
		return d, errors.Trace(err)
	}
	d.SetMysqlDuration(dur)
	return
}

type periodAddFunctionClass struct {
	baseFunctionClass
}

func (c *periodAddFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinPeriodAddSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinPeriodAddSig struct {
	baseBuiltinFunc
}

// eval evals a builtinPeriodAddSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_period-add
func (b *builtinPeriodAddSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}

	if args[0].IsNull() || args[1].IsNull() {
		return d, errors.Trace(err)
	}

	sc := b.ctx.GetSessionVars().StmtCtx
	period, err := args[0].ToInt64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	//Check zero
	if period <= 0 {
		d.SetInt64(0)
		return
	}

	y, m := getYearAndMonth(period)

	months, err := args[1].ToInt64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	sum := time.Month(m + months)
	// TODO: Consider time_zone variable.
	t := time.Date(int(y), sum, 1, 0, 0, 0, 0, time.Local)

	ret := int64(t.Year())*100 + int64(t.Month())
	d.SetInt64(ret)
	return
}

type periodDiffFunctionClass struct {
	baseFunctionClass
}

func (c *periodDiffFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinPeriodDiffSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinPeriodDiffSig struct {
	baseBuiltinFunc
}

// eval evals a builtinPeriodDiffSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_period-diff
func (b *builtinPeriodDiffSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}

	if args[0].IsNull() || args[1].IsNull() {
		return
	}

	sc := b.ctx.GetSessionVars().StmtCtx
	period, err := args[0].ToInt64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	period2, err := args[1].ToInt64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	d.SetInt64(getMonth(period) - getMonth(period2))
	return
}

func getYearAndMonth(period int64) (int64, int64) {
	y := period / 100
	m := period % 100

	// YYMM, 00-69 year: 2000-2069
	// YYMM, 70-99 year: 1970-1999
	if y <= 69 {
		y += 2000
	} else if y <= 99 {
		y += 1900
	}
	return y, m

}

func getMonth(period int64) int64 {
	if period == 0 {
		return int64(0)
	}
	y, m := getYearAndMonth(period)
	return y*12 + m - 1

}

type quarterFunctionClass struct {
	baseFunctionClass
}

func (c *quarterFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinQuarterSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinQuarterSig struct {
	baseBuiltinFunc
}

// eval evals a builtinQuarterSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_quarter
func (b *builtinQuarterSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}

	d, err = builtinMonth(args, b.ctx)
	if err != nil || d.IsNull() {
		return d, errors.Trace(err)
	}

	mon := int(d.GetInt64())
	switch mon {
	case 0:
		// An undocumented behavior of the mysql implementation
		d.SetInt64(0)
	case 1, 2, 3:
		d.SetInt64(1)
	case 4, 5, 6:
		d.SetInt64(2)
	case 7, 8, 9:
		d.SetInt64(3)
	case 10, 11, 12:
		d.SetInt64(4)
	default:
		d.SetNull()
		return d, errors.Errorf("no quarter for invalid month: %d.", mon)
	}
	return
}

type secToTimeFunctionClass struct {
	baseFunctionClass
}

func (c *secToTimeFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinSecToTimeSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinSecToTimeSig struct {
	baseBuiltinFunc
}

// eval evals a builtinSecToTimeSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_sec-to-time
func (b *builtinSecToTimeSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	if args[0].IsNull() {
		return
	}

	var (
		hour          int64
		minute        int64
		second        int64
		demical       float64
		secondDemical float64
		negative      string
	)

	sc := b.ctx.GetSessionVars().StmtCtx
	secondsFloat, err := args[0].ToFloat64(sc)
	if err != nil {
		if args[0].Kind() == types.KindString && types.ErrTruncated.Equal(err) {
			secondsFloat = float64(0)
		} else {
			return d, errors.Trace(err)
		}
	}

	if secondsFloat < 0 {
		negative = "-"
		secondsFloat = math.Abs(secondsFloat)
	}
	seconds := int64(secondsFloat)
	demical = secondsFloat - float64(seconds)

	hour = seconds / 3600
	if hour > 838 {
		hour = 838
		minute = 59
		second = 59
	} else {
		minute = seconds % 3600 / 60
		second = seconds % 60
	}
	secondDemical = float64(second) + demical

	var dur types.Duration
	fsp := types.MaxFsp
	if args[0].Kind() != types.KindString {
		sec, _ := args[0].ToString()
		secs := strings.Split(sec, ".")
		if len(secs) <= 1 {
			fsp = 0
		} else if len(secs[1]) < fsp {
			fsp = len(secs[1])
		}
	}
	dur, err = types.ParseDuration(fmt.Sprintf("%s%02d:%02d:%v", negative, hour, minute, secondDemical), fsp)
	if err != nil {
		return d, errors.Trace(err)
	}
	d.SetMysqlDuration(dur)
	return
}

type subTimeFunctionClass struct {
	baseFunctionClass
}

func (c *subTimeFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinSubTimeSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinSubTimeSig struct {
	baseBuiltinFunc
}

// eval evals a builtinSubTimeSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_subtime
func (b *builtinSubTimeSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	if args[0].IsNull() || args[1].IsNull() {
		return
	}
	var arg1 types.Duration
	switch args[1].Kind() {
	case types.KindMysqlDuration:
		arg1 = args[1].GetMysqlDuration()
	default:
		s, err := args[1].ToString()
		if err != nil {
			return d, errors.Trace(err)
		}
		if getFsp(s) == 0 {
			arg1, err = types.ParseDuration(s, 0)
		} else {
			arg1, err = types.ParseDuration(s, types.MaxFsp)
		}
		if err != nil {
			return d, errors.Trace(err)
		}

	}
	switch args[0].Kind() {
	case types.KindMysqlTime:
		arg0 := args[0].GetMysqlTime()
		arg1time, err := arg1.ConvertToTime(uint8(getFsp(arg1.String())))
		if err != nil {
			return d, errors.Trace(err)
		}
		tmpDuration := arg0.Sub(&arg1time)
		result, err := tmpDuration.ConvertToTime(arg0.Type)
		if err != nil {
			return d, errors.Trace(err)
		}
		d.SetMysqlTime(result)
	case types.KindMysqlDuration:
		arg0 := args[0].GetMysqlDuration()
		result, err := arg0.Sub(arg1)
		if err != nil {
			return d, errors.Trace(err)
		}
		d.SetMysqlDuration(result)
	default:
		ss, err := args[0].ToString()
		if err != nil {
			return d, errors.Trace(err)
		}
		if isDuration(ss) {
			return strDurationSubDuration(ss, arg1)
		}
		return strDatetimeSubDuration(ss, arg1)
	}
	return
}

type timeFormatFunctionClass struct {
	baseFunctionClass
}

func (c *timeFormatFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpString, tpDuration, tpString)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// worst case: formatMask=%r%r%r...%r, each %r takes 11 characters
	bf.tp.Flen = (args[1].GetType().Flen + 1) / 2 * 11
	sig := &builtinTimeFormatSig{baseStringBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinTimeFormatSig struct {
	baseStringBuiltinFunc
}

// evalString evals a builtinTimeFormatSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_time-format
func (b *builtinTimeFormatSig) evalString(row []types.Datum) (string, bool, error) {
	dur, isNull, err := b.args[0].EvalDuration(row, b.ctx.GetSessionVars().StmtCtx)
	// if err != nil, then dur is ZeroDuration, outputs 00:00:00 in this case which follows the behavior of mysql.
	if err != nil {
		log.Warnf("Expression.EvalDuration() in time_format() failed, due to %s", err.Error())
	}
	if isNull {
		return "", isNull, errors.Trace(err)
	}
	formatMask, isNull, err := b.args[1].EvalString(row, b.ctx.GetSessionVars().StmtCtx)
	if err != nil || isNull {
		return "", isNull, errors.Trace(err)
	}
	res, err := b.formatTime(dur, formatMask, b.ctx)
	return res, isNull, errors.Trace(err)
}

// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_time-format
func (b *builtinTimeFormatSig) formatTime(t types.Duration, formatMask string, ctx context.Context) (res string, err error) {
	t2 := types.Time{
		Time: types.FromDate(0, 0, 0, t.Hour(), t.Minute(), t.Second(), t.MicroSecond()),
		Type: mysql.TypeDate, Fsp: 0}

	str, err := t2.DateFormat(formatMask)
	return str, errors.Trace(err)
}

type timeToSecFunctionClass struct {
	baseFunctionClass
}

func (c *timeToSecFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinTimeToSecSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinTimeToSecSig struct {
	baseBuiltinFunc
}

// eval evals a builtinTimeToSecSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_time-to-sec
func (b *builtinTimeToSecSig) eval(row []types.Datum) (d types.Datum, err error) {
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}

	d, err = convertToDuration(b.ctx.GetSessionVars().StmtCtx, args[0], 0)
	if err != nil || d.IsNull() {
		return d, errors.Trace(err)
	}

	t := d.GetMysqlDuration()
	// TODO: select TIME_TO_SEC('-2:-2:-2') not handle well.
	if t.Compare(types.ZeroDuration) < 0 {
		d.SetInt64(int64(-1 * (t.Hour()*3600 + t.Minute()*60 + t.Second())))
	} else {
		d.SetInt64(int64(t.Hour()*3600 + t.Minute()*60 + t.Second()))
	}
	return
}

type timestampAddFunctionClass struct {
	baseFunctionClass
}

func (c *timestampAddFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpString, tpString, tpInt, tpTime)
	bf.tp = &types.FieldType{Tp: mysql.TypeString, Flen: mysql.MaxDatetimeWidthNoFsp, Decimal: types.UnspecifiedLength}
	if err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinTimestampAddSig{baseStringBuiltinFunc{bf}}
	return sig.setSelf(sig), nil

}

type builtinTimestampAddSig struct {
	baseStringBuiltinFunc
}

// evalString evals a builtinTimestampAddSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_timestampadd
func (b *builtinTimestampAddSig) evalString(row []types.Datum) (string, bool, error) {
	ctx := b.getCtx().GetSessionVars().StmtCtx
	unit, isNull, err := b.args[0].EvalString(row, ctx)
	if isNull || err != nil {
		return "", isNull, errors.Trace(err)
	}
	v, isNull, err := b.args[1].EvalInt(row, ctx)
	if isNull || err != nil {
		return "", isNull, errors.Trace(err)
	}
	arg, isNull, err := b.args[2].EvalTime(row, ctx)
	if isNull || err != nil {
		return "", isNull, errors.Trace(err)
	}
	tm1, err := arg.Time.GoTime(time.Local)
	if err != nil {
		return "", isNull, errors.Trace(err)
	}
	var tb time.Time
	fsp := types.DefaultFsp
	switch unit {
	case "MICROSECOND":
		tb = tm1.Add(time.Duration(v) * time.Microsecond)
		fsp = types.MaxFsp
	case "SECOND":
		tb = tm1.Add(time.Duration(v) * time.Second)
	case "MINUTE":
		tb = tm1.Add(time.Duration(v) * time.Minute)
	case "HOUR":
		tb = tm1.Add(time.Duration(v) * time.Hour)
	case "DAY":
		tb = tm1.AddDate(0, 0, int(v))
	case "WEEK":
		tb = tm1.AddDate(0, 0, 7*int(v))
	case "MONTH":
		tb = tm1.AddDate(0, int(v), 0)
	case "QUARTER":
		tb = tm1.AddDate(0, 3*int(v), 0)
	case "YEAR":
		tb = tm1.AddDate(int(v), 0, 0)
	default:
		return "", true, errors.Trace(types.ErrInvalidTimeFormat)
	}
	r := types.Time{Time: types.FromGoTime(tb), Type: mysql.TypeDatetime, Fsp: fsp}
	if err = r.Check(); err != nil {
		return "", true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	return r.String(), false, nil
}

type toDaysFunctionClass struct {
	baseFunctionClass
}

func (c *toDaysFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinToDaysSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinToDaysSig struct {
	baseIntBuiltinFunc
}

// evalInt evals a builtinToDaysSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_to-days
func (b *builtinToDaysSig) evalInt(row []types.Datum) (int64, bool, error) {
	sc := b.ctx.GetSessionVars().StmtCtx
	arg, isNull, err := b.args[0].EvalTime(row, sc)

	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	if err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	ret := types.TimestampDiff("DAY", types.ZeroDate, arg)
	if ret == 0 {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}
	return ret, false, nil
}

type toSecondsFunctionClass struct {
	baseFunctionClass
}

func (c *toSecondsFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	bf, err := newBaseBuiltinFuncWithTp(args, ctx, tpInt, tpTime)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinToSecondsSig{baseIntBuiltinFunc{bf}}
	return sig.setSelf(sig), nil
}

type builtinToSecondsSig struct {
	baseIntBuiltinFunc
}

// evalInt evals a builtinToSecondsSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_to-seconds
func (b *builtinToSecondsSig) evalInt(row []types.Datum) (int64, bool, error) {
	arg, isNull, err := b.args[0].EvalTime(row, b.getCtx().GetSessionVars().StmtCtx)
	if isNull || err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	if err != nil {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, err))
	}
	ret := types.TimestampDiff("SECOND", types.ZeroDate, arg)
	if ret == 0 {
		return 0, true, errors.Trace(handleInvalidTimeError(b.ctx, types.ErrInvalidTimeFormat))
	}
	return ret, false, nil
}

type utcTimeFunctionClass struct {
	baseFunctionClass
}

func (c *utcTimeFunctionClass) getFunction(args []Expression, ctx context.Context) (builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, errors.Trace(err)
	}
	sig := &builtinUTCTimeSig{newBaseBuiltinFunc(args, ctx)}
	return sig.setSelf(sig), nil
}

type builtinUTCTimeSig struct {
	baseBuiltinFunc
}

// eval evals a builtinUTCTimeSig.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_utc-time
func (b *builtinUTCTimeSig) eval(row []types.Datum) (d types.Datum, err error) {
	const utctimeFormat = "15:04:05.000000"
	args, err := b.evalArgs(row)
	if err != nil {
		return d, errors.Trace(err)
	}
	fsp := 0
	sc := b.ctx.GetSessionVars().StmtCtx
	if len(args) == 1 && !args[0].IsNull() {
		if fsp, err = checkFsp(sc, args[0]); err != nil {
			d.SetNull()
			return d, errors.Trace(err)
		}
	}
	d.SetString(time.Now().UTC().Format(utctimeFormat))
	return convertToDuration(b.ctx.GetSessionVars().StmtCtx, d, fsp)
}
