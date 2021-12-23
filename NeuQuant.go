package neuquant

import "math"

const (
	netsize = 256 // number of colours used

	// four primes near 500 - assume no image has a length so large
	// that it is divisible by all four primes
	prime1 = 499
	prime2 = 491
	prime3 = 487
	prime4 = 503

	minpicturebytes = 3 * prime4 //minimum size for input image

	// Program Skeleton ---------------- [select samplefac in range 1..30] [read
	// image from input file] pic = (unsigned char*) malloc(3*width*height);
	// initnet(pic,3*width*height,samplefac); learn(); unbiasnet(); [write
	// output image header, using writecolourmap(f)] inxbuild(); write output
	// image using inxsearch(b,g,r)

	// Network Definitions -------------------
	maxnetpos    = netsize - 1
	netbiasshift = 4   // bias for colour values
	ncycles      = 100 // no. of learning cycles

	// defs for freq and bias
	intbiasshift = 16 // bias for fractions
	intbias      = 1 << intbiasshift
	gammashift   = 10 // gamma = 1024
	// gamma        = 1 << gammashift
	betashift = 10
	beta      = intbias >> betashift // beta = 1/1024
	betagamma = intbias << (gammashift - betashift)

	// defs for decreasing radius factor
	initrad         = netsize >> 3
	radiusbiasshift = 6
	radiusbias      = 1 << radiusbiasshift
	initradius      = initrad * radiusbias // and decreases by a
	radiusdec       = 30                   // factor of 1/30 each cycle

	// defs for decreasing alpha factor
	alphabiasshift = 10 // alpha starts at 1.0
	initalpha      = 1 << alphabiasshift

	// radbias and alpharadbias used for radpower calculation
	radbiasshift   = 8
	radbias        = 1 << radbiasshift
	alpharadbshift = alphabiasshift + radbiasshift
	alpharadbias   = 1 << alpharadbshift
)

type NeuQuant struct {
	alphadec    int             // biased by 10 bits
	thepicture  []byte          // the input image itself
	lengthcount int             // lengthcount = H*W*3
	samplefac   int             // sampling factor 1..30
	network     [netsize][4]int // the network itself - [netsize][4]
	netindex    [256]int        // for network lookup - really 256
	bias        [netsize]int    // bias arrays for learning
	freq        [netsize]int    // freq arrays for learning
	radpower    [initrad]int    // radpower for precomputation
}

// NewNeuQuant Initialise network in range (0,0,0) to (255,255,255) and set parameters
func NewNeuQuant(thepic []byte, len, sample int) *NeuQuant {
	n := NeuQuant{
		thepicture:  thepic,
		lengthcount: len,
		samplefac:   sample,
	}

	for i := 0; i < netsize; i++ {
		v := (i << (netbiasshift + 8)) / netsize
		n.network[i][0] = v
		n.network[i][1] = v
		n.network[i][2] = v
		n.freq[i] = intbias / netsize // 1/netsize
		n.bias[i] = 0
	}

	n.learn()
	n.unbiasnet()
	n.inxbuild()

	return &n
}

func (n *NeuQuant) ColorMap() []byte {
	ret := make([]byte, 3*netsize)
	index := [netsize]int{}
	for i := 0; i < netsize; i++ {
		index[n.network[i][3]] = i
	}
	k := 0
	for i := 0; i < netsize; i++ {
		j := index[i]
		ret[k] = byte(n.network[j][0])
		ret[k+1] = byte(n.network[j][1])
		ret[k+2] = byte(n.network[j][2])
		k += 3
	}
	return ret
}

// inxbuild Insertion sort of network and building of netindex[0..255] (to do after unbias)
func (n *NeuQuant) inxbuild() {
	previouscol, startpos := 0, 0
	for i := 0; i < netsize; i++ {
		smallpos := i
		smallval := n.network[i][1] // index on g
		// find smallest in i..netsize-1
		for j := i + 1; j < netsize; j++ {
			if n.network[j][1] < smallval { // index on g
				smallpos = j
				smallval = n.network[j][1] // index on g
			}
		}
		// swap p (i) and q (smallpos) entries
		if i != smallpos {
			n.network[i][0], n.network[smallpos][0] = n.network[smallpos][0], n.network[i][0]
			n.network[i][1], n.network[smallpos][1] = n.network[smallpos][1], n.network[i][1]
			n.network[i][2], n.network[smallpos][2] = n.network[smallpos][2], n.network[i][2]
			n.network[i][3], n.network[smallpos][3] = n.network[smallpos][3], n.network[i][3]
		}
		// smallval entry is now in position i
		if smallval != previouscol {
			n.netindex[previouscol] = (startpos + i) >> 1
			for j := previouscol + 1; j < smallval; j++ {
				n.netindex[j] = i
			}
			previouscol = smallval
			startpos = i
		}
		n.netindex[previouscol] = (startpos + maxnetpos) >> 1
		for j := previouscol + 1; j < 256; j++ {
			n.netindex[j] = maxnetpos /* really 256 */
		}
	}
}

// learn Main Learning Loop ------------------
func (n *NeuQuant) learn() {
	if n.lengthcount < minpicturebytes {
		n.samplefac = 1
	}
	n.alphadec = 30 + ((n.samplefac - 1) / 3)
	p := n.thepicture
	pix := 0
	lim := n.lengthcount
	samplepixels := n.lengthcount / (3 * n.samplefac)
	delta := samplepixels / ncycles
	alpha := initalpha
	radius := initradius
	rad := radius >> radiusbiasshift
	// if rad <= 1 {
	//     rad = 0
	// }
	for i := 0; i < rad; i++ {
		n.radpower[i] = alpha * (((rad*rad - i*i) * radbias) / (rad * rad))
	}

	// fprintf(stderr,"beginning 1D learning: initial radius=%d\n", rad);

	var step int
	if n.lengthcount < minpicturebytes {
		step = 3
	} else if (n.lengthcount % prime1) != 0 {
		step = 3 * prime1
	} else {
		if (n.lengthcount % prime2) != 0 {
			step = 3 * prime2
		} else {
			if (n.lengthcount % prime3) != 0 {
				step = 3 * prime3
			} else {
				step = 3 * prime4
			}
		}
	}

	i := 0
	for i < samplepixels {
		b := (int(p[pix+0]) & 0xff) << netbiasshift
		g := (int(p[pix+1]) & 0xff) << netbiasshift
		r := (int(p[pix+2]) & 0xff) << netbiasshift
		j := n.contest(b, g, r)
		n.altersingle(alpha, j, b, g, r)
		if rad != 0 {
			n.alterneigh(rad, j, b, g, r) /* alter neighbours */
		}

		pix += step
		if pix >= lim {
			pix -= n.lengthcount
		}

		i++
		if delta == 0 {
			delta = 1
		}
		if i%delta == 0 {
			alpha -= alpha / n.alphadec
			radius -= radius / radiusdec
			rad = radius >> radiusbiasshift
			if rad <= 1 {
				rad = 0
			}
			for j = 0; j < rad; j++ {
				n.radpower[j] = alpha * (((rad*rad - j*j) * radbias) / (rad * rad))
			}
		}
	}
}

// Map Search for BGR values 0..255 (after net is unbiased) and return colour index
func (n *NeuQuant) Map(b, g, r int) int {
	bestd := 1000 // biggest possible dist is 256*3
	best := -1
	i := n.netindex[g] // index on g
	j := i - 1         // start at netindex[g] and work outwards

	for i < netsize || j >= 0 {
		if i < netsize {
			p := &n.network[i]
			dist := p[1] - g
			if dist >= bestd {
				i = netsize
			} else {
				i++
				if dist < 0 {
					dist = -dist
				}
				a := p[0] - b
				if a < 0 {
					a = -a
				}
				dist += a
				if dist < bestd {
					a = p[2] - r
					if a < 0 {
						a = -a
					}
					dist += a
					if dist < bestd {
						bestd = dist
						best = p[3]
					}
				}
			}
		}
		if j >= 0 {
			p := &n.network[j]
			dist := g - p[1] // inx key - reverse dif
			if dist >= bestd {
				j = -1 // stop iter
			} else {
				j--
				if dist < 0 {
					dist = -dist
				}
				a := p[0] - b
				if a < 0 {
					a = -a
				}
				dist += a
				if dist < bestd {
					a = p[2] - r
					if a < 0 {
						a = -a
					}
					dist += a
					if dist < bestd {
						bestd = dist
						best = p[3]
					}
				}
			}
		}
	}
	return best
}

// unbiasnet network to give byte values 0..255 and record position i to prepare for sort
func (n *NeuQuant) unbiasnet() {
	for i := 0; i < netsize; i++ {
		n.network[i][0] >>= netbiasshift
		n.network[i][1] >>= netbiasshift
		n.network[i][2] >>= netbiasshift
		n.network[i][3] = i // record colour no
	}
}

func (n *NeuQuant) alterneigh(rad, i, b, g, r int) {
	lo := i - rad
	if lo < -1 {
		lo = -1
	}
	hi := i + rad
	if hi > netsize {
		hi = netsize
	}

	j := i + 1
	k := i - 1
	m := 1
	for (j < hi) || (k > lo) {
		a := n.radpower[m]
		m++
		if j < hi {
			n.network[j][0] -= (a * (n.network[j][0] - b)) / alpharadbias
			n.network[j][1] -= (a * (n.network[j][1] - g)) / alpharadbias
			n.network[j][2] -= (a * (n.network[j][2] - r)) / alpharadbias
			j++
		}
		if k > lo {
			n.network[k][0] -= (a * (n.network[k][0] - b)) / alpharadbias
			n.network[k][1] -= (a * (n.network[k][1] - g)) / alpharadbias
			n.network[k][2] -= (a * (n.network[k][2] - r)) / alpharadbias
			k--
		}
	}
}

// Move neuron i towards biased (b,g,r) by factor alpha
func (n *NeuQuant) altersingle(alpha, i, b, g, r int) {
	n.network[i][0] -= (alpha * (n.network[i][0] - b)) / initalpha
	n.network[i][1] -= (alpha * (n.network[i][1] - g)) / initalpha
	n.network[i][2] -= (alpha * (n.network[i][2] - r)) / initalpha
}

// Search for biased BGR values ----------------------------
func (n *NeuQuant) contest(b, g, r int) int {
	// finds closest neuron (min dist) and updates freq
	// finds best neuron (min dist-bias) and returns position
	//
	// for frequently chosen neurons, freq[i] is high and bias[i] is negative
	// bias[i] = gamma*((1/netsize)-freq[i])

	bestd := math.MaxInt32
	bestbiasd := bestd
	bestpos := -1
	bestbiaspos := bestpos
	for i := 0; i < netsize; i++ {
		dist := n.network[i][0] - b
		if dist < 0 {
			dist = -dist
		}
		a := n.network[i][1] - g
		if a < 0 {
			a = -a
		}
		dist += a
		a = n.network[i][2] - r
		if a < 0 {
			a = -a
		}
		dist += a
		if dist < bestd {
			bestd = dist
			bestpos = i
		}
		biasdist := dist - ((n.bias[i]) >> (intbiasshift - netbiasshift))
		if biasdist < bestbiasd {
			bestbiasd = biasdist
			bestbiaspos = i
		}
		betafreq := n.freq[i] >> betashift
		n.freq[i] -= betafreq
		n.bias[i] += betafreq << gammashift
	}
	n.freq[bestpos] += beta
	n.bias[bestpos] -= betagamma
	return bestbiaspos
}
