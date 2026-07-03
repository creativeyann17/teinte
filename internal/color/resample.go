package color

// ResampleChannel linearly interpolates channel c of the ramp to n
// entries. X11 CRTCs report their own gamma table size (256, 1024, ...)
// while the ramp is always built at 256; this bridges the two.
func (r *Ramp) ResampleChannel(c, n int) []uint16 {
	out := make([]uint16, n)
	if n == 0 {
		return out
	}
	if n == 1 {
		out[0] = r[c][0]
		return out
	}
	for i := range n {
		pos := float64(i) * (RampSize - 1) / float64(n-1)
		lo := int(pos)
		hi := lo + 1
		if hi >= RampSize {
			hi = RampSize - 1
		}
		frac := pos - float64(lo)
		out[i] = uint16(float64(r[c][lo])*(1-frac) + float64(r[c][hi])*frac + 0.5)
	}
	return out
}
