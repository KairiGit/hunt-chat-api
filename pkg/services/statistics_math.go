package services

import (
	"math"
	"sort"
)

// studentTCDF computes the CDF of Student's t at x with df degrees of freedom.
// Uses regularized incomplete beta relation for accuracy.
func studentTCDF(x, df float64) float64 {
	// For x=0, CDF=0.5
	if x == 0 {
		return 0.5
	}
	// Relation: CDF = 0.5 + x * Gamma((v+1)/2) * 2F1(...) / (sqrt(v*pi) * Gamma(v/2))
	// We'll use the incomplete beta representation:
	// For t>0: CDF = 1 - 0.5*I_{v/(v+t^2)}(v/2, 1/2)
	// For t<0: CDF = 0.5*I_{v/(v+t^2)}(v/2, 1/2)
	v := df
	t2 := x * x
	z := v / (v + t2)
	ib := regularizedIncompleteBeta(0.5*v, 0.5, z)
	if x > 0 {
		return 1 - 0.5*ib
	}
	return 0.5 * ib
}

// regularizedIncompleteBeta returns I_x(a,b).
// We implement a simple continued fraction via Lentz's algorithm for the incomplete beta function ratio.
func regularizedIncompleteBeta(a, b, x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	// Use symmetry to improve convergence
	bt := math.Exp(lgamma(a+b) - lgamma(a) - lgamma(b) + a*math.Log(x) + b*math.Log(1-x))
	var ib float64
	if x < (a+1)/(a+b+2) {
		ib = bt * betacf(a, b, x) / a
	} else {
		ib = 1 - bt*betacf(b, a, 1-x)/b
	}
	return ib
}

// betacf computes the continued fraction for incomplete beta using Lentz's algorithm.
func betacf(a, b, x float64) float64 {
	const maxIter = 200
	const eps = 3e-7
	const fpmin = 1e-30
	am := 1.0
	bm := 1.0
	az := 1.0
	qab := a + b
	qap := a + 1
	qam := a - 1
	bz := 1 - qab*x/qap
	var em, tem, d, ap, bp, app, bpp, aold float64
	for m := 1; m <= maxIter; m++ {
		em = float64(m)
		tem = em + em
		d = em * (b - em) * x / ((qam + tem) * (a + tem))
		ap = az + d*am
		bp = bz + d*bm
		d = -(a + em) * (qab + em) * x / ((a + tem) * (qap + tem))
		app = ap + d*az
		bpp = bp + d*bz
		am = ap / bpp
		bm = bp / bpp
		az = app / bpp
		bz = 1.0
		aold = az
		if math.Abs((az-aold)/az) < eps {
			return az
		}
		if math.Abs(bpp) < fpmin {
			bpp = fpmin
		}
	}
	return az
}

// lgamma wrapper for math.Lgamma returning sign-less log gamma
func lgamma(x float64) float64 {
	l, _ := math.Lgamma(x)
	return l
}

// olsRSS computes the residual sum of squares of OLS regression y ~ X
func olsRSS(y []float64, X [][]float64) (float64, error) {
	n := len(y)
	if n == 0 {
		return 0, nil
	}
	k := len(X)
	if k == 0 {
		var ss float64
		for _, yi := range y {
			ss += yi * yi
		}
		return ss, nil
	}
	// Check dimensions
	for i := 0; i < k; i++ {
		if len(X[i]) != n {
			return 0, nil
		}
	}
	// Build X'X and X'y
	XtX := make([][]float64, k)
	for i := 0; i < k; i++ {
		XtX[i] = make([]float64, k)
		for j := 0; j < k; j++ {
			var sum float64
			for t := 0; t < n; t++ {
				sum += X[i][t] * X[j][t]
			}
			XtX[i][j] = sum
		}
	}
	Xty := make([]float64, k)
	for i := 0; i < k; i++ {
		var sum float64
		for t := 0; t < n; t++ {
			sum += X[i][t] * y[t]
		}
		Xty[i] = sum
	}
	// solve for coefficients
	beta, err := solveSymmetric(XtX, Xty)
	if err != nil {
		return 0, err
	}
	// compute residuals
	var rss float64
	for t := 0; t < n; t++ {
		var pred float64
		for i := 0; i < k; i++ {
			pred += beta[i] * X[i][t]
		}
		res := y[t] - pred
		rss += res * res
	}
	return rss, nil
}

// solveSymmetric solves A*x=b for symmetric positive definite A by Cholesky
func solveSymmetric(A [][]float64, b []float64) ([]float64, error) {
	n := len(A)
	if n == 0 {
		return nil, nil
	}
	for _, row := range A {
		if len(row) != n {
			return nil, nil
		}
	}
	if len(b) != n {
		return nil, nil
	}
	// copy A to L
	L := make([][]float64, n)
	for i := 0; i < n; i++ {
		L[i] = make([]float64, n)
		copy(L[i], A[i])
	}
	// Cholesky decomposition
	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			var sum float64
			for k := 0; k < j; k++ {
				sum += L[i][k] * L[j][k]
			}
			if i == j {
				val := L[i][i] - sum
				if val <= 0 {
					return nil, nil
				}
				L[i][j] = math.Sqrt(val)
			} else {
				if L[j][j] == 0 {
					return nil, nil
				}
				L[i][j] = (L[i][j] - sum) / L[j][j]
			}
		}
		for j := i + 1; j < n; j++ {
			L[i][j] = 0
		}
	}
	// Forward substitution
	y := make([]float64, n)
	for i := 0; i < n; i++ {
		var sum float64
		for j := 0; j < i; j++ {
			sum += L[i][j] * y[j]
		}
		y[i] = (b[i] - sum) / L[i][i]
	}
	// Back substitution
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		var sum float64
		for j := i + 1; j < n; j++ {
			sum += L[j][i] * x[j]
		}
		x[i] = (y[i] - sum) / L[i][i]
	}
	return x, nil
}

// fDistSurvival computes P(F > f) for F ~ F(d1, d2)
func fDistSurvival(f, d1, d2 float64) float64 {
	if f <= 0 {
		return 1.0
	}
	// F(d1,d2) ~ d1/(d1+d2*F) * Beta(d1/2, d2/2)
	x := d1 * f / (d1*f + d2)
	// survival = 1 - I_x(d1/2, d2/2) = I_{1-x}(d2/2, d1/2)
	return regularizedIncompleteBeta(d2/2, d1/2, 1-x)
}

// AdjustPValuesBH applies Benjamini-Hochberg FDR correction to a slice of p-values.
func (s *StatisticsService) AdjustPValuesBH(pvals []float64) []float64 {
	n := len(pvals)
	type kv struct {
		p float64
		i int
	}
	arr := make([]kv, n)
	for i, p := range pvals {
		arr[i] = kv{p: p, i: i}
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].p < arr[j].p })
	adj := make([]float64, n)
	var prev float64 = 1.0
	for i := n - 1; i >= 0; i-- {
		rank := float64(i + 1)
		val := arr[i].p * float64(n) / rank
		if val > prev {
			val = prev
		}
		if val > 1 {
			val = 1
		}
		adj[i] = val
		prev = val
	}
	// restore original order
	out := make([]float64, n)
	for idx, a := range adj {
		out[arr[idx].i] = a
	}
	return out
}

// FirstDifference transforms series to first differences to remove trend.
func (s *StatisticsService) FirstDifference(vals []float64) []float64 {
	if len(vals) < 2 {
		return vals
	}
	out := make([]float64, 0, len(vals)-1)
	for i := 1; i < len(vals); i++ {
		out = append(out, vals[i]-vals[i-1])
	}
	return out
}

// Detrend by subtracting linear trend.
func (s *StatisticsService) Detrend(vals []float64) []float64 {
	n := len(vals)
	if n < 2 {
		return vals
	}
	// simple linear regression on index
	xs := make([]float64, n)
	for i := 0; i < n; i++ {
		xs[i] = float64(i + 1)
	}
	reg, err := s.PerformLinearRegression(xs, vals)
	if err != nil {
		return vals
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = vals[i] - (reg.Slope*xs[i] + reg.Intercept)
	}
	return out
}

// calculateMean calculates the mean of a slice of float64 values
// calculateMean パッケージ内部用のヘルパー関数：平均値を計算
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateStandardDeviation パッケージ内部用のヘルパー関数：標準偏差を計算
func calculateStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := calculateMean(values)
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	return math.Sqrt(sumSquaredDiff / float64(len(values)))
}
