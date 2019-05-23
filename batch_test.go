// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package ach

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/moov-io/base"
)

// batch should never be used directly.
func mockBatch() *Batch {
	mockBatch := &Batch{}
	mockBatch.SetHeader(mockBatchHeader())
	mockBatch.AddEntry(mockEntryDetail())
	if err := mockBatch.build(); err != nil {
		panic(err)
	}
	return mockBatch
}

// Batch with mismatched TraceNumber ODFI
func mockBatchInvalidTraceNumberODFI() *Batch {
	mockBatch := &Batch{}
	mockBatch.SetHeader(mockBatchHeader())
	mockBatch.AddEntry(mockEntryDetailInvalidTraceNumberODFI())
	return mockBatch
}

// EntryDetail with mismatched TraceNumber ODFI
func mockEntryDetailInvalidTraceNumberODFI() *EntryDetail {
	entry := NewEntryDetail()
	entry.TransactionCode = CheckingCredit
	entry.SetRDFI("121042882")
	entry.DFIAccountNumber = "123456789"
	entry.Amount = 100000000
	entry.IndividualName = "Wade Arnold"
	entry.SetTraceNumber("9928272", 1)
	entry.IdentificationNumber = "ABC##jvkdjfuiwn"
	entry.Category = CategoryForward
	return entry
}

// Batch with no entries
func mockBatchNoEntry() *Batch {
	mockBatch := &Batch{}
	mockBatch.SetHeader(mockBatchHeader())
	return mockBatch
}

// Invalid SEC CODE BatchHeader
func mockBatchInvalidSECHeader() *BatchHeader {
	bh := NewBatchHeader()
	bh.ServiceClassCode = CreditsOnly
	bh.StandardEntryClassCode = "NIL"
	bh.CompanyName = "ACME Corporation"
	bh.CompanyIdentification = "123456789"
	bh.CompanyEntryDescription = "PAYROLL"
	bh.EffectiveEntryDate = time.Now().AddDate(0, 0, 1).Format("060102") // YYMMDD
	bh.ODFIIdentification = "123456789"
	return bh
}

// TestBatch__UnmarshalJSON reads an example File (with Batches) and attempts to unmarshal it as JSON
func TestBatch__UnmarshalJSON(t *testing.T) {
	// Make sure we don't panic with nil in the mix
	var batch *Batch
	if err := batch.UnmarshalJSON(nil); err != nil && !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatal(err)
	}

	// Read file, convert to JSON
	fd, err := os.Open(filepath.Join("test", "ach-pos-read", "pos-debit.ach"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewReader(fd).Read()
	if err != nil {
		t.Fatal(err)
	}

	bs, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}

	// Read as JSON
	file, err := FileFromJSON(bs)
	if err != nil {
		t.Fatal(err)
	}
	if file == nil {
		t.Error("file == nil")
	}

	if v := file.Header.FileCreationDate; v != "180614" {
		t.Errorf("got FileCreationDate of %q", v)
	}
	if v := file.Header.FileCreationTime; v != "0000" {
		t.Errorf("got FileCreationTime of %q", v)
	}
}

// Test cases that apply to all batch types
// testBatchNumberMismatch validates BatchNumber mismatch
func testBatchNumberMismatch(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.GetControl().BatchNumber = 2
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchHeaderControlEquality(1, 2)) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchNumberMismatch tests validating BatchNumber mismatch
func TestBatchNumberMismatch(t *testing.T) {
	testBatchNumberMismatch(t)
}

// BenchmarkBatchNumberMismatch benchmarks validating BatchNumber mismatch
func BenchmarkBatchNumberMismatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchNumberMismatch(b)
	}
}

// testCreditBatchIsBatchAmount validates Batch TotalCreditEntryDollarAmount
func testCreditBatchIsBatchAmount(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.SetHeader(mockBatchHeader())
	e1 := mockBatch.GetEntries()[0]
	e1.TransactionCode = CheckingCredit
	e1.Amount = 100
	e2 := mockEntryDetail()
	e2.TransactionCode = CheckingCredit
	e2.Amount = 100
	// replace last 2 of TraceNumber
	e2.TraceNumber = e1.TraceNumber[:13] + "10"
	mockBatch.AddEntry(e2)
	if err := mockBatch.build(); err != nil {
		t.Errorf("%T: %s", err, err)
	}

	mockBatch.GetControl().TotalCreditEntryDollarAmount = 1
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchCalculatedControlEquality(200, 1)) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestCreditBatchIsBatchAmount test validating Batch TotalCreditEntryDollarAmount
func TestCreditBatchIsBatchAmount(t *testing.T) {
	testCreditBatchIsBatchAmount(t)
}

// BenchmarkCreditBatchIsBatchAmount benchmarks Batch TotalCreditEntryDollarAmount
func BenchmarkCreditBatchIsBatchAmount(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testCreditBatchIsBatchAmount(b)
	}

}

// testSavingsBatchIsBatchAmount validates Batch TotalDebitEntryDollarAmount
func testSavingsBatchIsBatchAmount(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.SetHeader(mockBatchHeader())
	e1 := mockBatch.GetEntries()[0]
	e1.TransactionCode = SavingsCredit
	e1.Amount = 100
	e2 := mockEntryDetail()
	e2.TransactionCode = SavingsDebit
	e2.Amount = 100
	// replace last 2 of TraceNumber
	e2.TraceNumber = e1.TraceNumber[:13] + "10"

	mockBatch.AddEntry(e2)
	if err := mockBatch.build(); err != nil {
		t.Errorf("%T: %s", err, err)
	}

	mockBatch.GetControl().TotalDebitEntryDollarAmount = 1
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchCalculatedControlEquality(200, 1)) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestSavingsBatchIsBatchAmount tests validating Batch TotalDebitEntryDollarAmount
func TestSavingsBatchIsBatchAmount(t *testing.T) {
	testSavingsBatchIsBatchAmount(t)
}

// BenchmarkSavingsBatchIsBatchAmount benchmarks validating Batch TotalDebitEntryDollarAmount
func BenchmarkSavingsBatchIsBatchAmount(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testSavingsBatchIsBatchAmount(b)
	}
}

func testBatchIsEntryHash(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.GetControl().EntryHash = 1
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchCalculatedControlEquality(12104288, 1)) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchIsEntryHash(t *testing.T) {
	testBatchIsEntryHash(t)
}

func BenchmarkBatchIsEntryHash(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchIsEntryHash(b)
	}
}

func testBatchDNEMismatch(t testing.TB) {
	bh := mockBatchHeader()
	bh.StandardEntryClassCode = DNE
	mockBatch := mockBatch()
	mockBatch.SetHeader(bh)
	ed := mockBatch.GetEntries()[0]
	ed.AddAddenda05(mockAddenda05())
	ed.AddAddenda05(mockAddenda05())
	mockBatch.build()

	mockBatch.GetHeader().OriginatorStatusCode = 1
	mockBatch.GetEntries()[0].TransactionCode = CheckingPrenoteCredit
	err := mockBatch.verify()
	if !base.Match(err, ErrBatchOriginatorDNE) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchDNEMismatch(t *testing.T) {
	testBatchDNEMismatch(t)
}

func BenchmarkBatchDNEMismatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchDNEMismatch(b)
	}
}

func TestBatch__DNEOriginatorCheck(t *testing.T) {
	bh := mockBatchHeader()
	bh.OriginatorStatusCode = 1
	bh.StandardEntryClassCode = PPD

	batch := mockBatch()
	batch.SetHeader(bh)

	if err := batch.isOriginatorDNE(); err != nil {
		t.Errorf("%T: %s", err, err)
	}
}

func testBatchTraceNumberNotODFI(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.GetEntries()[0].SetTraceNumber("12345678", 1)
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchTraceNumberNotODFI("12104288", "12345678")) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchTraceNumberNotODFI(t *testing.T) {
	testBatchTraceNumberNotODFI(t)
}

func BenchmarkBatchTraceNumberNotODFI(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchTraceNumberNotODFI(b)
	}
}

func testBatchEntryCountEquality(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.SetHeader(mockBatchHeader())
	e := mockEntryDetail()
	a := mockAddenda05()
	e.AddAddenda05(a)
	mockBatch.AddEntry(e)
	if err := mockBatch.build(); err != nil {
		t.Errorf("%T: %s", err, err)
	}

	mockBatch.GetControl().EntryAddendaCount = 1
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchCalculatedControlEquality(3, 1)) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchEntryCountEquality(t *testing.T) {
	testBatchEntryCountEquality(t)
}

func BenchmarkBatchEntryCountEquality(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchEntryCountEquality(b)
	}
}

func testBatchAddendaIndicator(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.GetEntries()[0].AddAddenda05(mockAddenda05())
	mockBatch.GetEntries()[0].AddendaRecordIndicator = 0
	mockBatch.GetControl().EntryAddendaCount = 2
	err := mockBatch.verify()
	if !base.Match(err, ErrBatchAddendaIndicator) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchAddendaIndicator(t *testing.T) {
	testBatchAddendaIndicator(t)
}

func BenchmarkBatchAddendaIndicator(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchAddendaIndicator(b)
	}
}

func testBatchIsAddendaSeqAscending(t testing.TB) {
	mockBatch := mockBatch()
	ed := mockBatch.GetEntries()[0]
	ed.AddAddenda05(mockAddenda05())
	ed.AddAddenda05(mockAddenda05())
	mockBatch.build()
	mockBatch.Entries[0].AddendaRecordIndicator = 1
	mockBatch.GetEntries()[0].Addenda05[0].SequenceNumber = 2
	mockBatch.GetEntries()[0].Addenda05[1].SequenceNumber = 1
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchAscending(2, 1)) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchIsAddendaSeqAscending(t *testing.T) {
	testBatchIsAddendaSeqAscending(t)
}
func BenchmarkBatchIsAddendaSeqAscending(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchIsAddendaSeqAscending(b)
	}
}

func testBatchIsSequenceAscending(t testing.TB) {
	mockBatch := mockBatch()
	e3 := mockEntryDetail()
	e3.TraceNumber = "1"
	mockBatch.AddEntry(e3)
	mockBatch.GetControl().EntryAddendaCount = 2
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchAscending(121042880000001, 1)) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchIsSequenceAscending(t *testing.T) {
	testBatchIsSequenceAscending(t)
}

func BenchmarkBatchIsSequenceAscending(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchIsSequenceAscending(b)
	}
}

func testBatchAddendaTraceNumber(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.GetEntries()[0].AddAddenda05(mockAddenda05())
	if err := mockBatch.build(); err != nil {
		t.Errorf("%T: %s", err, err)
	}
	mockBatch.Entries[0].AddendaRecordIndicator = 1
	mockBatch.GetEntries()[0].Addenda05[0].EntryDetailSequenceNumber = 99
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchAscending("1", "1")) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchAddendaTraceNumber(t *testing.T) {
	testBatchAddendaTraceNumber(t)
}

func BenchmarkBatchAddendaTraceNumber(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchAddendaTraceNumber(b)
	}
}

// testNewBatchDefault validates error for NewBatch if invalid SEC Code
func testNewBatchDefault(t testing.TB) {
	_, err := NewBatch(mockBatchInvalidSECHeader())

	if err != NewErrFileUnknownSEC("NIL") {
		t.Errorf("%T: %s", err, err)
	}
}

// TestNewBatchDefault test validating error for NewBatch if invalid SEC Code
func TestNewBatchDefault(t *testing.T) {
	testNewBatchDefault(t)
}

// BenchmarkNewBatchDefault benchmarks validating error for NewBatch if
// invalid SEC Code
func BenchmarkNewBatchDefault(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testNewBatchDefault(b)
	}
}

// testBatchCategory validates Batch Category
func testBatchCategory(t testing.TB) {
	mockBatch := mockBatch()
	// Add a Addenda Return to the mock batch
	entry := mockEntryDetail()
	entry.Addenda99 = mockAddenda99()
	entry.Category = CategoryReturn
	mockBatch.AddEntry(entry)

	if err := mockBatch.build(); err != nil {
		t.Errorf("%T: %s", err, err)
	}

	if mockBatch.Category() != CategoryReturn {
		t.Errorf("Addenda99 added to batch and category is %s", mockBatch.Category())
	}
}

// TestBatchCategory tests validating Batch Category
func TestBatchCategory(t *testing.T) {
	testBatchCategory(t)
}

// BenchmarkBatchCategory benchmarks validating Batch Category
func BenchmarkBatchCategory(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchCategory(b)
	}
}

//  testBatchCategoryForwardReturn validates Category based on EntryDetail
func testBatchCategoryForwardReturn(t testing.TB) {
	mockBatch := mockBatch()
	// Add a Addenda Return to the mock batch
	entry := mockEntryDetail()
	entry.Addenda99 = mockAddenda99()
	entry.Category = CategoryReturn
	// replace last 2 of TraceNumber
	entry.TraceNumber = entry.TraceNumber[:13] + "10"
	entry.AddendaRecordIndicator = 1
	mockBatch.AddEntry(entry)
	if err := mockBatch.build(); err != nil {
		t.Errorf("%T: %s", err, err)
	}
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchCategory("Return", "Forward")) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchCategoryForwardReturn tests validating Category based on EntryDetail
func TestBatchCategoryForwardReturn(t *testing.T) {
	testBatchCategoryForwardReturn(t)
}

//  BenchmarkBatchCategoryForwardReturn benchmarks validating Category based on EntryDetail
func BenchmarkBatchCategoryForwardReturn(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchCategoryForwardReturn(b)
	}
}

// Don't over write a batch trace number when building if it already exists
func testBatchTraceNumberExists(t testing.TB) {
	mockBatch := mockBatch()
	entry := mockEntryDetail()
	traceBefore := entry.TraceNumberField()
	mockBatch.AddEntry(entry)
	mockBatch.build()
	traceAfter := mockBatch.GetEntries()[1].TraceNumberField()
	if traceBefore != traceAfter {
		t.Errorf("Trace number was set to %v before batch.build and is now %v\n", traceBefore, traceAfter)
	}
}

func TestBatchTraceNumberExists(t *testing.T) {
	testBatchTraceNumberExists(t)
}

func BenchmarkBatchTraceNumberExists(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchTraceNumberExists(b)
	}
}

func testBatchFieldInclusion(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.Header.ODFIIdentification = ""
	err := mockBatch.verify()
	if !base.Match(err, ErrConstructor) {
		t.Errorf("%T: %s", err, err)
	}
}

func TestBatchFieldInclusion(t *testing.T) {
	testBatchFieldInclusion(t)
}

func BenchmarkBatchFieldInclusion(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchFieldInclusion(b)
	}
}

// testBatchInvalidTraceNumberODFI validates TraceNumberODFI
func testBatchInvalidTraceNumberODFI(t testing.TB) {
	mockBatch := mockBatchInvalidTraceNumberODFI()
	if err := mockBatch.build(); err != nil {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchInvalidTraceNumberODFI tests validating TraceNumberODFI
func TestBatchInvalidTraceNumberODFI(t *testing.T) {
	testBatchInvalidTraceNumberODFI(t)
}

// BenchmarkBatchInvalidTraceNumberODFI benchmarks validating TraceNumberODFI
func BenchmarkBatchInvalidTraceNumberODFI(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchInvalidTraceNumberODFI(b)
	}
}

// testBatchNoEntry validates error for a batch with no entries
func testBatchNoEntry(t testing.TB) {
	mockBatch := mockBatchNoEntry()
	err := mockBatch.build()
	if !base.Match(err, ErrBatchNoEntries) {
		t.Errorf("%T: %s", err, err)
	}

	// test verify
	err = mockBatch.verify()
	if !base.Match(err, ErrBatchNoEntries) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchNoEntry tests validating error for a batch with no entries
func TestBatchNoEntry(t *testing.T) {
	testBatchNoEntry(t)
}

// BenchmarkBatchNoEntry benchmarks validating error for a batch with no entries
func BenchmarkBatchNoEntry(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchNoEntry(b)
	}
}

// testBatchControl validates BatchControl ODFIIdentification
func testBatchControl(t testing.TB) {
	mockBatch := mockBatch()
	mockBatch.Control.ODFIIdentification = ""
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchHeaderControlEquality("12104288", "")) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchControl tests validating BatchControl ODFIIdentification
func TestBatchControl(t *testing.T) {
	testBatchControl(t)
}

// BenchmarkBatchControl benchmarks validating BatchControl ODFIIdentification
func BenchmarkBatchControl(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testBatchControl(b)
	}
}

// testIATBatch validates an IAT batch returns an error for batch
func testIATBatch(t testing.TB) {
	bh := NewBatchHeader()
	bh.ServiceClassCode = CreditsOnly
	bh.StandardEntryClassCode = IAT
	bh.CompanyName = "ACME Corporation"
	bh.CompanyIdentification = "123456789"
	bh.CompanyEntryDescription = "PAYROLL"
	bh.EffectiveEntryDate = time.Now().AddDate(0, 0, 1).Format("060102") // YYMMDD
	bh.ODFIIdentification = "123456789"

	_, err := NewBatch(bh)

	if err != ErrFileIATSEC {
		t.Errorf("%T: %s", err, err)
	}
}

// TestIATBatch tests validating an IAT batch returns an error for batch
func TestIATBatch(t *testing.T) {
	testIATBatch(t)
}

// BenchmarkIATBatch benchmarks validating an IAT batch returns an error for batch
func BenchmarkIATBatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testIATBatch(b)
	}
}

// TestBatchADVInvalidServiceClassCode validates ServiceClassCode
func TestBatchADVInvalidServiceClassCode(t *testing.T) {
	mockBatch := mockBatchADV()
	if err := mockBatch.Create(); err != nil {
		t.Fatal(err)
	}
	mockBatch.ADVControl.ServiceClassCode = CreditsOnly
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchHeaderControlEquality("280", "220")) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchADVInvalidODFIIdentification validates ODFIIdentification
func TestBatchADVInvalidODFIIdentification(t *testing.T) {
	mockBatch := mockBatchADV()
	if err := mockBatch.Create(); err != nil {
		t.Fatal(err)
	}
	mockBatch.ADVControl.ODFIIdentification = "231380104"
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchHeaderControlEquality("12104288", "231380104")) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchADVInvalidBatchNumber validates BatchNumber
func TestBatchADVInvalidBatchNumber(t *testing.T) {
	mockBatch := mockBatchADV()
	if err := mockBatch.Create(); err != nil {
		t.Fatal(err)
	}
	mockBatch.ADVControl.BatchNumber = 2
	err := mockBatch.verify()
	if !base.Match(err, NewErrBatchHeaderControlEquality("1", "2")) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchADVEntryAddendaCount validates EntryAddendaCount
func TestBatchADVInvalidEntryAddendaCount(t *testing.T) {
	mockBatch := mockBatchADV()
	if err := mockBatch.Create(); err != nil {
		t.Fatal(err)
	}
	mockBatch.ADVControl.EntryAddendaCount = CheckingCredit
	err := mockBatch.Validate()
	if !base.Match(err, NewErrBatchCalculatedControlEquality(1, 22)) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchADVTotalDebitEntryDollarAmount validates TotalDebitEntryDollarAmount
func TestBatchADVInvalidTotalDebitEntryDollarAmount(t *testing.T) {
	mockBatch := mockBatchADV()
	mockBatch.GetADVEntries()[0].TransactionCode = DebitForCreditsOriginated
	if err := mockBatch.Create(); err != nil {
		t.Fatal(err)
	}
	mockBatch.ADVControl.TotalDebitEntryDollarAmount = 2200
	err := mockBatch.Validate()
	if !base.Match(err, NewErrBatchCalculatedControlEquality(50000, 2200)) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchADVTotalCreditEntryDollarAmount validates TotalCreditEntryDollarAmount
func TestBatchADVInvalidTotalCreditEntryDollarAmount(t *testing.T) {
	mockBatch := mockBatchADV()
	if err := mockBatch.Create(); err != nil {
		t.Fatal(err)
	}
	mockBatch.ADVControl.TotalCreditEntryDollarAmount = 2200
	err := mockBatch.Validate()
	if !base.Match(err, NewErrBatchCalculatedControlEquality(50000, 2200)) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchADVEntryHash validates EntryHash
func TestBatchADVInvalidEntryHash(t *testing.T) {
	mockBatch := mockBatchADV()
	if err := mockBatch.Create(); err != nil {
		t.Fatal(err)
	}
	mockBatch.ADVControl.EntryHash = 2200233
	err := mockBatch.Validate()
	if !base.Match(err, NewErrBatchCalculatedControlEquality(23138010, 2200233)) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchAddenda98InvalidAddendaRecordIndicator validates AddendaRecordIndicator
func TestBatchAddenda98InvalidAddendaRecordIndicator(t *testing.T) {
	mockBatch := mockBatchCOR()
	mockBatch.GetEntries()[0].AddendaRecordIndicator = 0
	err := mockBatch.Create()
	if !base.Match(err, ErrBatchAddendaIndicator) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchAddenda02InvalidAddendaRecordIndicator validates AddendaRecordIndicator
func TestBatchAddenda02InvalidAddendaRecordIndicator(t *testing.T) {
	mockBatch := mockBatchPOS()
	mockBatch.GetEntries()[0].AddendaRecordIndicator = 0
	err := mockBatch.Create()
	if !base.Match(err, ErrBatchAddendaIndicator) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchADVCategory validates Category
func TestBatchADVCategory(t *testing.T) {
	mockBatch := mockBatchADV()

	entryOne := NewADVEntryDetail()
	entryOne.TransactionCode = CreditForDebitsOriginated
	entryOne.SetRDFI("231380104")
	entryOne.DFIAccountNumber = "744-5678-99"
	entryOne.Amount = 50000
	entryOne.AdviceRoutingNumber = "121042882"
	entryOne.FileIdentification = "FILE1"
	entryOne.ACHOperatorData = ""
	entryOne.IndividualName = "Name"
	entryOne.DiscretionaryData = ""
	entryOne.AddendaRecordIndicator = 0
	entryOne.ACHOperatorRoutingNumber = "01100001"
	entryOne.JulianDay = 50
	entryOne.SequenceNumber = 1
	entryOne.Category = CategoryReturn

	mockBatch.AddADVEntry(entryOne)
	err := mockBatch.Create()
	if !base.Match(err, NewErrBatchCategory("Return", "Forward")) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchDishonoredReturnsCategory validates Category for Returns
func TestBatchDishonoredReturnsCategory(t *testing.T) {
	entry := NewEntryDetail()
	entry.TransactionCode = CheckingDebit
	entry.SetRDFI("121042882")
	entry.DFIAccountNumber = "744-5678-99"
	entry.Amount = 25000
	entry.IdentificationNumber = "45689033"
	entry.IndividualName = "Wade Arnold"
	entry.SetTraceNumber(mockBatchPOSHeader().ODFIIdentification, 1)
	entry.DiscretionaryData = "01"
	entry.AddendaRecordIndicator = 1
	entry.Category = CategoryDishonoredReturn

	addenda99 := mockAddenda99()
	addenda99.ReturnCode = "R68"
	addenda99.AddendaInformation = "Untimely Return"
	entry.Addenda99 = addenda99

	entryOne := NewEntryDetail()
	entryOne.TransactionCode = CheckingDebit
	entryOne.SetRDFI("121042882")
	entryOne.DFIAccountNumber = "744-5678-99"
	entryOne.Amount = 23000
	entryOne.IdentificationNumber = "45689033"
	entryOne.IndividualName = "Adam Decaf"
	entryOne.SetTraceNumber(mockBatchPOSHeader().ODFIIdentification, 1)
	entryOne.DiscretionaryData = "01"
	entryOne.AddendaRecordIndicator = 1
	entryOne.Category = CategoryReturn

	addenda99One := mockAddenda99()
	addenda99One.ReturnCode = "R68"
	addenda99One.AddendaInformation = "Untimely Return"
	entryOne.Addenda99 = addenda99One

	posHeader := NewBatchHeader()
	posHeader.ServiceClassCode = DebitsOnly
	posHeader.StandardEntryClassCode = POS
	posHeader.CompanyName = "Payee Name"
	posHeader.CompanyIdentification = "231380104"
	posHeader.CompanyEntryDescription = "ACH POS"
	posHeader.ODFIIdentification = "23138010"

	batch := NewBatchPOS(posHeader)
	batch.SetHeader(posHeader)
	batch.AddEntry(entry)
	batch.AddEntry(entryOne)

	err := batch.Create()
	if !base.Match(err, NewErrBatchCategory("Return", "DishonoredReturn")) {
		t.Errorf("%T: %s", err, err)
	}
}

// TestBatchConvertBatchType validates ConvertBatchType
func TestBatchConvertBatchType(t *testing.T) {
	mockBatchACK := mockBatchACK()
	convertedACK := ConvertBatchType(mockBatchACK.Batch)
	if reflect.TypeOf(convertedACK) != reflect.TypeOf(mockBatchACK) {
		t.Error("ACK batch type is not converted correctly")
	}
	mockBatchADV := mockBatchADV()
	convertedADV := ConvertBatchType(mockBatchADV.Batch)
	if reflect.TypeOf(convertedADV) != reflect.TypeOf(mockBatchADV) {
		t.Error("ADV batch type is not converted correctly")
	}
	mockBatchARC := mockBatchARC()
	convertedARC := ConvertBatchType(mockBatchARC.Batch)
	if reflect.TypeOf(convertedARC) != reflect.TypeOf(mockBatchARC) {
		t.Error("ARC batch type is not converted correctly")
	}
	mockBatchATX := mockBatchATX()
	convertedATX := ConvertBatchType(mockBatchATX.Batch)
	if reflect.TypeOf(convertedATX) != reflect.TypeOf(mockBatchATX) {
		t.Error("ATX batch type is not converted correctly")
	}
	mockBatchBOC := mockBatchBOC()
	convertedBOC := ConvertBatchType(mockBatchBOC.Batch)
	if reflect.TypeOf(convertedBOC) != reflect.TypeOf(mockBatchBOC) {
		t.Error("BOC batch type is not converted correctly")
	}
	mockBatchCCD := mockBatchCCD()
	convertedCCD := ConvertBatchType(mockBatchCCD.Batch)
	if reflect.TypeOf(convertedCCD) != reflect.TypeOf(mockBatchCCD) {
		t.Error("CCD batch type is not converted correctly")
	}
	mockBatchCIE := mockBatchCIE()
	convertedCIE := ConvertBatchType(mockBatchCIE.Batch)
	if reflect.TypeOf(convertedCIE) != reflect.TypeOf(mockBatchCIE) {
		t.Error("CIE batch type is not converted correctly")
	}
	mockBatchCOR := mockBatchCOR()
	convertedCOR := ConvertBatchType(mockBatchCOR.Batch)
	if reflect.TypeOf(convertedCOR) != reflect.TypeOf(mockBatchCOR) {
		t.Error("COR batch type is not converted correctly")
	}
	mockBatchCTX := mockBatchCTX()
	convertedCTX := ConvertBatchType(mockBatchCTX.Batch)
	if reflect.TypeOf(convertedCTX) != reflect.TypeOf(mockBatchCTX) {
		t.Error("CTX batch type is not converted correctly")
	}
	mockBatchDNE := mockBatchDNE()
	convertedDNE := ConvertBatchType(mockBatchDNE.Batch)
	if reflect.TypeOf(convertedDNE) != reflect.TypeOf(mockBatchDNE) {
		t.Error("DNE batch type is not converted correctly")
	}
	mockBatchENR := mockBatchENR()
	convertedENR := ConvertBatchType(mockBatchENR.Batch)
	if reflect.TypeOf(convertedENR) != reflect.TypeOf(mockBatchENR) {
		t.Error("ENR batch type is not converted correctly")
	}
	mockBatchMTE := mockBatchMTE()
	convertedMTE := ConvertBatchType(mockBatchMTE.Batch)
	if reflect.TypeOf(convertedMTE) != reflect.TypeOf(mockBatchMTE) {
		t.Error("MTE batch type is not converted correctly")
	}
	mockBatchPOP := mockBatchPOP()
	convertedPOP := ConvertBatchType(mockBatchPOP.Batch)
	if reflect.TypeOf(convertedPOP) != reflect.TypeOf(mockBatchPOP) {
		t.Error("POP batch type is not converted correctly")
	}
	mockBatchPOS := mockBatchPOS()
	convertedPOS := ConvertBatchType(mockBatchPOS.Batch)
	if reflect.TypeOf(convertedPOS) != reflect.TypeOf(mockBatchPOS) {
		t.Error("POS batch type is not converted correctly")
	}
	mockBatchPPD := mockBatchPPD()
	convertedPPD := ConvertBatchType(mockBatchPPD.Batch)
	if reflect.TypeOf(convertedPPD) != reflect.TypeOf(mockBatchPPD) {
		t.Error("PPD batch type is not converted correctly")
	}
	mockBatchRCK := mockBatchRCK()
	convertedRCK := ConvertBatchType(mockBatchRCK.Batch)
	if reflect.TypeOf(convertedRCK) != reflect.TypeOf(mockBatchRCK) {
		t.Error("RCK batch type is not converted correctly")
	}
	mockBatchSHR := mockBatchSHR()
	convertedSHR := ConvertBatchType(mockBatchSHR.Batch)
	if reflect.TypeOf(convertedSHR) != reflect.TypeOf(mockBatchSHR) {
		t.Error("SHR batch type is not converted correctly")
	}
	mockBatchTEL := mockBatchTEL()
	convertedTEL := ConvertBatchType(mockBatchTEL.Batch)
	if reflect.TypeOf(convertedTEL) != reflect.TypeOf(mockBatchTEL) {
		t.Error("TEL batch type is not converted correctly")
	}
	mockBatchTRC := mockBatchTRC()
	convertedTRC := ConvertBatchType(mockBatchTRC.Batch)
	if reflect.TypeOf(convertedTRC) != reflect.TypeOf(mockBatchTRC) {
		t.Error("TRC batch type is not converted correctly")
	}
	mockBatchTRX := mockBatchTRX()
	convertedTRX := ConvertBatchType(mockBatchTRX.Batch)
	if reflect.TypeOf(convertedTRX) != reflect.TypeOf(mockBatchTRX) {
		t.Error("TRX batch type is not converted correctly")
	}
	mockBatchWEB := mockBatchWEB()
	convertedWEB := ConvertBatchType(mockBatchWEB.Batch)
	if reflect.TypeOf(convertedWEB) != reflect.TypeOf(mockBatchWEB) {
		t.Error("WEB batch type is not converted correctly")
	}
	mockBatchXCK := mockBatchXCK()
	convertedXCK := ConvertBatchType(mockBatchXCK.Batch)
	if reflect.TypeOf(convertedXCK) != reflect.TypeOf(mockBatchXCK) {
		t.Error("XCK batch type is not converted correctly")
	}
}

func TestBatch__Equal(t *testing.T) {
	testFile := func(t *testing.T) *File {
		t.Helper()
		fd, err := os.Open(filepath.Join("test", "testdata", "ppd-debit.ach"))
		if err != nil {
			t.Fatal(err)
		}
		defer fd.Close()
		file, err := NewReader(fd).Read()
		if err != nil {
			t.Fatal(err)
		}
		return &file
	}

	firstBatch := testFile(t).Batches[0]

	// Let's check and ensure equality
	secondBatch := testFile(t).Batches[0]
	if !firstBatch.Equal(secondBatch) {
		t.Fatal("identical .Equal failed, uhh")
	}

	// nil cases
	var b *Batch
	if b.Equal(secondBatch) || secondBatch.Equal(nil) {
		t.Fatalf("b.Equal(secondBatch)=%v secondBatch.Equal(nil)=%v", b.Equal(secondBatch), secondBatch.Equal(nil))
	}

	// Now change each field in .Equal and see
	secondBatch = testFile(t).Batches[0]
	secondBatch.GetHeader().ServiceClassCode = 1
	if firstBatch.Equal(secondBatch) {
		t.Error("changed ServiceClassCode, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetHeader().StandardEntryClassCode = "ZZZ"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed StandardEntryClassCode, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetHeader().CompanyName = "foo"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed CompanyName, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetHeader().CompanyIdentification = "new company"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed CompanyIdentification, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetHeader().EffectiveEntryDate = "1111"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed EffectiveEntryDate, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetHeader().ODFIIdentification = "12"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed ODFIIdentification, expected not equal")
	}

	// Check differences in EntryDetail
	secondBatch = testFile(t).Batches[0]
	secondBatch.GetEntries()[0].TransactionCode = 1
	if firstBatch.Equal(secondBatch) {
		t.Error("changed TransactionCode, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetEntries()[0].RDFIIdentification = "41"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed RDFIIdentification, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetEntries()[0].DFIAccountNumber = "542"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed DFIAccountNumber, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetEntries()[0].Amount = 1
	if firstBatch.Equal(secondBatch) {
		t.Error("changed Amount, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetEntries()[0].IdentificationNumber = "99"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed IdentificationNumber, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetEntries()[0].IndividualName = "jane doe"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed IndividualName, expected not equal")
	}

	secondBatch = testFile(t).Batches[0]
	secondBatch.GetEntries()[0].DiscretionaryData = "other info"
	if firstBatch.Equal(secondBatch) {
		t.Error("changed DiscretionaryData, expected not equal")
	}

	// Add another EntryDetail and make sure we fail
	secondBatch = testFile(t).Batches[0]
	secondBatch.AddEntry(secondBatch.GetEntries()[0])
	if firstBatch.Equal(secondBatch) {
		t.Error("added EntryDetail, expected not equal")
	}
}
