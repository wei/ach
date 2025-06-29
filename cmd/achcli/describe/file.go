// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package describe

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/moov-io/ach"
	"github.com/moov-io/ach/cmd/achcli/describe/mask"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

type Opts struct {
	mask.Options

	PrettyAmounts bool
}

func File(ww io.Writer, file *ach.File, opts *Opts) {
	if file == nil {
		return
	}
	if opts == nil {
		opts = &Opts{}
	}

	// Mask the file
	file = mask.File(file, opts.Options)

	w := tabwriter.NewWriter(ww, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fh, fc := file.Header, file.Control

	// FileHeader
	fmt.Fprintln(w, "  Origin\tOriginName\tDestination\tDestinationName\tFileCreationDate\tFileCreationTime")
	fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n", fh.ImmediateOriginField(), fh.ImmediateOriginNameField(), fh.ImmediateDestinationField(), fh.ImmediateDestinationNameField(), fh.FileCreationDateField(), fh.FileCreationTimeField())

	// Batches
	for i := range file.Batches {
		fmt.Fprintln(w, "\n  BatchNumber\tSECCode\tServiceClassCode\tCompanyName\tDiscretionaryData\tIdentification\tEntryDescription\tEffectiveEntryDate\tDescriptiveDate")

		bh := file.Batches[i].GetHeader()
		if bh != nil {
			fmt.Fprintf(w, "  %s\t%s\t%d %s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				bh.BatchNumberField(),
				bh.StandardEntryClassCode,
				bh.ServiceClassCode,
				serviceClassCodes[bh.ServiceClassCode],
				bh.CompanyNameField(),
				bh.CompanyDiscretionaryDataField(),
				bh.CompanyIdentificationField(),
				bh.CompanyEntryDescriptionField(),
				bh.EffectiveEntryDateField(),
				bh.CompanyDescriptiveDateField(),
			)
		}

		entries := file.Batches[i].GetEntries()
		for j := range entries {
			fmt.Fprintln(w, "\n    TransactionCode\tRDFIIdentification\tAccountNumber\tAmount\tName\tIdentificationNumber\tTraceNumber\tCategory")

			e := entries[j]

			amount := formatAmount(opts.PrettyAmounts, e.Amount)

			fmt.Fprintf(w, "    %d %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", e.TransactionCode, transactionCodes[e.TransactionCode], e.RDFIIdentificationField(), e.DFIAccountNumberField(), amount, e.IndividualNameField(), e.IdentificationNumberField(), e.TraceNumberField(), e.Category)

			dumpAddenda02(w, e.Addenda02)
			for a := range e.Addenda05 {
				if a == 0 {
					fmt.Fprintln(w, "\n      Addenda05")
				}
				dumpAddenda05(w, file.Batches[i], e.Addenda05[a], opts)
			}
			dumpAddenda98(w, opts, e.Addenda98)
			dumpAddenda99(w, e.Addenda99)
			dumpAddenda99Dishonored(w, e.Addenda99Dishonored)
			dumpAddenda99Contested(w, e.Addenda99Contested)
		}

		bc := file.Batches[i].GetControl()
		if bc != nil {
			fmt.Fprintln(w, "\n  ServiceClassCode\tEntryAddendaCount\tEntryHash\tTotalDebits\tTotalCredits\tMACCode\tODFIIdentification\tBatchNumber")

			debitTotal := formatAmount(opts.PrettyAmounts, bc.TotalDebitEntryDollarAmount)
			creditTotal := formatAmount(opts.PrettyAmounts, bc.TotalCreditEntryDollarAmount)
			fmt.Fprintf(w, "  %d %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				bc.ServiceClassCode, serviceClassCodes[bh.ServiceClassCode], bc.EntryAddendaCountField(), bc.EntryHashField(), debitTotal, creditTotal, bc.MessageAuthenticationCodeField(), bc.ODFIIdentificationField(), bc.BatchNumberField())
		}
	}

	// IATBatches
	for i := range file.IATBatches {
		iatBatch := file.IATBatches[i]
		bh := iatBatch.GetHeader()
		if bh != nil {
			fmt.Fprintln(w, "\n  BatchNumber\tSECCode\tServiceClassCode\tIATIndicator\tDestinationCountryCode\tFE Indicator\tFE ReferenceIndicator\tFE Reference\tCompanyEntryDescription")
			fmt.Fprintf(w, "  %s\t%s\t%d %s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				bh.BatchNumberField(),
				bh.StandardEntryClassCode,
				bh.ServiceClassCode,
				serviceClassCodes[bh.ServiceClassCode],
				bh.IATIndicatorField(),
				bh.ISODestinationCountryCodeField(),
				bh.ForeignExchangeIndicatorField(),
				bh.ForeignExchangeReferenceIndicatorField(),
				bh.ForeignExchangeReferenceField(),
				bh.CompanyEntryDescriptionField(),
			)

			fmt.Fprintln(w, "\n    OriginatorIdentification\tISOOriginatingCurrencyCode\tISODestinationCurrencyCode\tODFIIdentification\tEffectiveEntryDate\tOriginatorStatusCode")
			fmt.Fprintf(w, "    %s\t%s\t%s\t%s\t%s\t%d\n",
				bh.OriginatorIdentificationField(),
				bh.ISOOriginatingCurrencyCodeField(),
				bh.ISODestinationCurrencyCodeField(),
				bh.ODFIIdentificationField(),
				bh.EffectiveEntryDateField(),
				bh.OriginatorStatusCode,
			)
		}

		entries := iatBatch.GetEntries()
		for j := range entries {
			fmt.Fprintln(w, "\n    TransactionCode\tRDFIIdentification\tAccountNumber\tAmount\tAddendaRecords\tTraceNumber\tCategory")

			e := entries[j]
			amount := formatAmount(opts.PrettyAmounts, e.Amount)
			fmt.Fprintf(w, "    %d %s\t%s\t%s\t%s\t%s\t%s\t%s\n", e.TransactionCode, transactionCodes[e.TransactionCode], e.RDFIIdentificationField(), e.DFIAccountNumberField(), amount, e.AddendaRecordsField(), e.TraceNumberField(), e.Category)

			dumpAddenda10(w, e.Addenda10)
			dumpAddenda11(w, e.Addenda11)
			dumpAddenda12(w, e.Addenda12)
			dumpAddenda13(w, e.Addenda13)
			dumpAddenda14(w, e.Addenda14)
			dumpAddenda15(w, e.Addenda15)
			dumpAddenda16(w, e.Addenda16)

			for i := range e.Addenda17 {
				dumpAddenda17(w, e.Addenda17[i])
			}
			for i := range e.Addenda18 {
				dumpAddenda18(w, e.Addenda18[i])
			}

			dumpAddenda98(w, opts, e.Addenda98)
			dumpAddenda99(w, e.Addenda99)
		}

		bc := iatBatch.GetControl()
		if bc != nil {
			fmt.Fprintln(w, "\n  ServiceClassCode\tEntryAddendaCount\tEntryHash\tTotalDebits\tTotalCredits\tMACCode\tODFIIdentification\tBatchNumber")

			debitTotal := formatAmount(opts.PrettyAmounts, bc.TotalDebitEntryDollarAmount)
			creditTotal := formatAmount(opts.PrettyAmounts, bc.TotalCreditEntryDollarAmount)
			fmt.Fprintf(w, "  %d %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				bc.ServiceClassCode, serviceClassCodes[bh.ServiceClassCode], bc.EntryAddendaCountField(), bc.EntryHashField(), debitTotal, creditTotal, bc.MessageAuthenticationCodeField(), bc.ODFIIdentificationField(), bc.BatchNumberField())
		}
	}

	// FileControl
	fmt.Fprintln(w, "\n  BatchCount\tBlockCount\tEntryAddendaCount\tTotalDebitAmount\tTotalCreditAmount")

	debitTotal := formatAmount(opts.PrettyAmounts, fc.TotalDebitEntryDollarAmountInFile)
	creditTotal := formatAmount(opts.PrettyAmounts, fc.TotalCreditEntryDollarAmountInFile)
	fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", fc.BatchCountField(), fc.BlockCountField(), fc.EntryAddendaCountField(), debitTotal, creditTotal)
}

// formatAmount can optionally convert an integer into a human readable amount
func formatAmount(prettyAmounts bool, amt int) string {
	if !prettyAmounts {
		return strconv.Itoa(amt)
	}

	printer := message.NewPrinter(language.Und)
	formatter := number.Decimal(float64(amt)/100.0, number.MinFractionDigits(2))
	return printer.Sprint(formatter)
}

func dumpAddenda02(w *tabwriter.Writer, a *ach.Addenda02) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "\n      Addenda02")
	fmt.Fprintln(w, "      ReferenceInfoOne\tReferenceInfoTwo\tTerminalIdentification\tTransactionSerial\tDate\tAuthCodeOrExires\tLocation\tCity\tState\tTraceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		a.ReferenceInformationOneField(), a.ReferenceInformationTwoField(), a.TerminalIdentificationCodeField(), a.TransactionSerialNumberField(),
		a.TransactionDateField(), a.AuthorizationCodeOrExpireDateField(), a.TerminalLocationField(), a.TerminalCityField(), a.TerminalStateField(), a.TraceNumberField())
}

func dumpAddenda99Dishonored(w *tabwriter.Writer, a *ach.Addenda99Dishonored) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "\n      Dishonored Addenda99")
	fmt.Fprintln(w, "      Dis. ReturnCode\tOrig. TraceNumber\tRDFI Identification\tTraceNumber\tSettlementDate\tReturnCode\tAddendaInformation\tTraceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		a.DishonoredReturnReasonCodeField(), a.OriginalEntryTraceNumberField(), a.OriginalReceivingDFIIdentificationField(), a.ReturnTraceNumberField(),
		a.ReturnSettlementDateField(), a.ReturnReasonCodeField(), a.AddendaInformationField(), a.TraceNumberField())
}

func dumpAddenda99Contested(w *tabwriter.Writer, a *ach.Addenda99Contested) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "\n      Contested Dishonored Addenda99")
	fmt.Fprintln(w, "      ContestedReturnCode\tOrig. TraceNumber\tOrig Date Returned\tOrig. RDFIIdentification\tOrig. SettlementDate\tReturnTraceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\n",
		a.ContestedReturnCodeField(), a.OriginalEntryTraceNumberField(), a.DateOriginalEntryReturnedField(), a.OriginalReceivingDFIIdentificationField(),
		a.OriginalSettlementDateField(), a.ReturnTraceNumberField())

	fmt.Fprintln(w, "      ReturnSettlementDate\tReturnReasonCode\tDishonoredTraceNumber\tDishonoredSettlementDate\tDishonoredReasonCode\tTraceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\n",
		a.ReturnSettlementDateField(), a.ReturnReasonCodeField(), a.DishonoredReturnTraceNumberField(), a.DishonoredReturnSettlementDateField(), a.DishonoredReturnReasonCodeField(), a.TraceNumberField())
}

func dumpAddenda05(w *tabwriter.Writer, batch ach.Batcher, a *ach.Addenda05, opts *Opts) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      PaymentRelatedInformation\tSequenceNumber\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\n", a.PaymentRelatedInformationField(), a.SequenceNumberField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda98(w *tabwriter.Writer, opts *Opts, a *ach.Addenda98) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "\n      Addenda98")
	fmt.Fprintln(w, "      ChangeCode\tOriginalTrace\tOriginalDFI\tCorrectedData\tTraceNumber")

	data := a.CorrectedData
	if opts.MaskCorrectedData {
		data = mask.Number(data)
	}

	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\n", a.ChangeCode, a.OriginalTraceField(), a.OriginalDFIField(), data, a.TraceNumberField())
}

func dumpAddenda99(w *tabwriter.Writer, a *ach.Addenda99) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "\n      Addenda99")
	fmt.Fprintln(w, "      ReturnCode\tOriginalTrace\tDateOfDeath\tOriginalDFI\tAddendaInformation\tTraceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\n", a.ReturnCode, a.OriginalTraceField(), a.DateOfDeathField(), a.OriginalDFIField(), a.AddendaInformationField(), a.TraceNumberField())
}

func dumpAddenda10(w *tabwriter.Writer, a *ach.Addenda10) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tTransactionTypeCode\tForeignPaymentAmount\tForeignTraceNumber\tName\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\n", a.TypeCode, a.TransactionTypeCode, a.ForeignPaymentAmountField(), a.ForeignTraceNumberField(), a.NameField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda11(w *tabwriter.Writer, a *ach.Addenda11) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tOriginatorName\tOriginatorStreetAddress\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\n", a.TypeCode, a.OriginatorNameField(), a.OriginatorStreetAddressField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda12(w *tabwriter.Writer, a *ach.Addenda12) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tOriginatorCityStateProvince\tOriginatorCountryPostalCode\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\n", a.TypeCode, a.OriginatorCityStateProvinceField(), a.OriginatorCountryPostalCodeField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda13(w *tabwriter.Writer, a *ach.Addenda13) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tODFIName\tODFIIDNumberQualifier\tODFIIdentification\tODFIBranchCountryCode\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\n", a.TypeCode, a.ODFINameField(), a.ODFIIDNumberQualifierField(), a.ODFIIdentificationField(), a.ODFIBranchCountryCodeField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda14(w *tabwriter.Writer, a *ach.Addenda14) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tRDFIName\tRDFIIDNumberQualifier\tRDFIIdentification\tRDFIBranchCountryCode\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\n", a.TypeCode, a.RDFINameField(), a.RDFIIDNumberQualifierField(), a.RDFIIdentificationField(), a.RDFIBranchCountryCodeField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda15(w *tabwriter.Writer, a *ach.Addenda15) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tReceiverIDNumber\tReceiverStreetAddress\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\n", a.TypeCode, a.ReceiverIDNumberField(), a.ReceiverStreetAddressField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda16(w *tabwriter.Writer, a *ach.Addenda16) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tReceiverCityStateProvince\tReceiverCountryPostalCode\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\n", a.TypeCode, a.ReceiverCityStateProvinceField(), a.ReceiverCountryPostalCodeField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda17(w *tabwriter.Writer, a *ach.Addenda17) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tPaymentRelatedInformation\tSequenceNumber\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\n", a.TypeCode, a.PaymentRelatedInformationField(), a.SequenceNumberField(), a.EntryDetailSequenceNumberField())
}

func dumpAddenda18(w *tabwriter.Writer, a *ach.Addenda18) {
	if a == nil {
		return
	}

	fmt.Fprintln(w, "      TypeCode\tForeignCorrespondentBankName\tForeignCorrespondentBankIDNumberQualifier\tForeignCorrespondentBankIDNumber\tForeignCorrespondentBankBranchCountryCode\tSequenceNumber\tEntryDetailSequenceNumber")
	fmt.Fprintf(w, "      %s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		a.TypeCode, a.ForeignCorrespondentBankNameField(), a.ForeignCorrespondentBankIDNumberQualifierField(), a.ForeignCorrespondentBankIDNumberField(),
		a.ForeignCorrespondentBankBranchCountryCodeField(), a.SequenceNumberField(), a.EntryDetailSequenceNumberField())
}
