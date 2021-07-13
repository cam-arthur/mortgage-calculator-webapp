package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type MortgageOptions struct {
	Mortgage                 float64
	InterestRate             float64
	Repayment                float64
	VoluntaryMonthly         float64
	VoluntaryFortnightly     float64
	VoluntaryWeekly          float64
	VoluntaryDaily           float64
	LumpSum                  float64
	LumpSumPayday            int
	LumpSumYearly            bool
	RepaymentDays            int
	IdealYears               int
	DaysInYear               int
	InterestRateYearlyChange float64
}

type MortgageStatsYearly struct {
	Year                  int     `json:"year"`
	MortgageAmount        float64 `json:"mortgageamount"`
	TotalInterestYear     float64 `json:"totalinterestyear"`
	TotalInterestLife     float64 `json:"totalinterestlife"`
	TotalRepaidYear       float64 `json:"totalrepaidyear"`
	TotalRepaidLife       float64 `json:"totalrepaidlife"`
	MandatoryPaymentsMade float64 `json:"mandatorypaymentsmade"`
}

type MortgageSummary struct {
	Results []MortgageStatsYearly `json:"results"`
}

func newRouter() *mux.Router {
	r := mux.NewRouter()
	staticFileDirectory := http.Dir("./assets/")
	staticFileHandler := http.StripPrefix("/home", http.FileServer(staticFileDirectory))
	r.PathPrefix("/home").Handler(staticFileHandler).Methods("GET")
	// r.PathPrefix("/charts/mortgagebreakdown").Handler(http.StripPrefix("/charts/mortgagebreakdown", http.Dir("./assets/mortgagebreakdown.html"))).Methods("GET")
	r.HandleFunc(`/getmortgagestats`, handlerRepaymentStats).Methods("POST")
	r.HandleFunc("/getsampledatafastrepayment", sampleDataHandlerFastRepayment).Methods("GET")
	r.HandleFunc("/getsampledatastandardrepayment", sampleDataHandlerStandardRepayment).Methods("GET")
	return r
}

func main() {
	// The router is now formed by calling the `newRouter` constructor function
	// that we defined above. The rest of the code stays the same
	r := newRouter()
	err := http.ListenAndServe(":6969", r)
	// err := http.ListenAndServe(":42069", r)
	if err != nil {
		panic(err.Error())
	}
}

func sampleDataHandlerFastRepayment(w http.ResponseWriter, r *http.Request) {
	w.Write(getMortgageStatsByte(MortgageOptions{230000, 2.49, 454, 0, 0, 250, 5, 10000, 60, true, 14, 100, 365, 0}))
}
func sampleDataHandlerStandardRepayment(w http.ResponseWriter, r *http.Request) {
	w.Write(getMortgageStatsByte(MortgageOptions{230000, 2.49, 454, 0, 0, 0, 0, 0, 0, true, 14, 100, 365, 0}))
}

func getMortgageStatsByte(mortgageOptions MortgageOptions) []byte {
	resultSetBytes, err := json.Marshal(calculateMortgage(mortgageOptions))
	check(err)
	return resultSetBytes
}

func handlerRepaymentStats(w http.ResponseWriter, r *http.Request) {
	var mortgageOptions MortgageOptions
	// fmt.Println(r.Body)
	err := json.NewDecoder(r.Body).Decode(&mortgageOptions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// fmt.Println(mortgageOptions)
	w.Write(getMortgageStatsByte(mortgageOptions))
}

func calculateInterest(interestRate, mortgageAmount, total float64) (float64, float64) {
	interestCharged := mortgageAmount * (1 + interestRate)
	return interestCharged, total + interestCharged - mortgageAmount
}

func calculateRepayments(mortgage, repayment, total float64) (float64, float64) {
	return mortgage - repayment, repayment + total
}

func calculateMortgage(mortgageOptions MortgageOptions) MortgageSummary {
	var yearlyMortgageResults MortgageSummary
	var mortgage = mortgageOptions.Mortgage
	var year = 0
	var totalInterest float64 = 0
	var totalRepaid float64 = 0
	var mandatoryPayments float64 = 0
	yearlyMortgageResults.Results = append(yearlyMortgageResults.Results, generateOutput(mortgage, 0, 0, mandatoryPayments, totalInterest, totalRepaid, year))
	currentDay := 1
	for mortgage > 0 {
		year++
		var totalInterestYear float64 = 0
		var totalRepaidYear float64 = 0
		for day := currentDay; day <= mortgageOptions.DaysInYear*year; day++ {
			dailyRate := ((mortgageOptions.InterestRate / 100) / float64(mortgageOptions.DaysInYear))
			if mortgage > 0 {
				mortgage, totalInterestYear = calculateInterest(dailyRate, mortgage, totalInterestYear)
				if day%mortgageOptions.RepaymentDays == 0 {
					mortgage, totalRepaidYear = calculateRepayments(mortgage, mortgageOptions.Repayment, totalRepaidYear)
					mandatoryPayments += mortgageOptions.Repayment
				}
				if day%30 == 0 && year <= mortgageOptions.IdealYears {
					mortgage, totalRepaidYear = calculateRepayments(mortgage, mortgageOptions.VoluntaryMonthly, totalRepaidYear)
				}
				if day%14 == 0 && year <= mortgageOptions.IdealYears {
					mortgage, totalRepaidYear = calculateRepayments(mortgage, mortgageOptions.VoluntaryFortnightly, totalRepaidYear)
				}
				if day%7 == 0 && year <= mortgageOptions.IdealYears {
					mortgage, totalRepaidYear = calculateRepayments(mortgage, mortgageOptions.VoluntaryWeekly, totalRepaidYear)
				}
				if day%1 == 0 && year <= mortgageOptions.IdealYears {
					mortgage, totalRepaidYear = calculateRepayments(mortgage, mortgageOptions.VoluntaryDaily, totalRepaidYear)
				}

				if (!mortgageOptions.LumpSumYearly && day == mortgageOptions.LumpSumPayday) || (mortgageOptions.LumpSumYearly && mortgageOptions.DaysInYear-(year*mortgageOptions.DaysInYear-day) == mortgageOptions.LumpSumPayday) {
					mortgage, totalRepaidYear = calculateRepayments(mortgage, mortgageOptions.LumpSum, totalRepaidYear)
				}
			}
			currentDay++
		}
		if mortgage < 0 {
			mortgage = 0
		}
		totalInterest += totalInterestYear
		totalRepaid += totalRepaidYear
		if mortgage > mortgageOptions.Mortgage {
			mortgage = 0
			yearlyMortgageResults.Results = append(yearlyMortgageResults.Results, generateOutput(mortgage, 0, 0, 0, 0, 0, year))
		} else {
			yearlyMortgageResults.Results = append(yearlyMortgageResults.Results, generateOutput(mortgage, totalInterestYear, totalRepaidYear, mandatoryPayments, totalInterest, totalRepaid, year))
		}

		mortgageOptions.InterestRate += mortgageOptions.InterestRateYearlyChange
	}
	return yearlyMortgageResults
}

func generateOutput(mortgage, totalInterestYear, totalRepaidYear, mandatoryPayments, totalInterest, totalRepaid float64, year int) MortgageStatsYearly {
	return MortgageStatsYearly{year, mortgage, totalInterestYear, totalInterest, totalRepaidYear, totalRepaid, mandatoryPayments}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
