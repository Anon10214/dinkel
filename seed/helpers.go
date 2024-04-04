package seed

import "math"

// BooleanWithProbability uses the passed seed to generate a boolean with the value "true" with the probability provided.
// Assumes that 0 <= probability <= 1
func (s *Seed) BooleanWithProbability(probability float64) bool {
	generatedInt := s.GetRandomPositiveInt64()
	return float64(math.MaxInt)*probability > float64(generatedInt)
}

// RandomBoolean returns true with probability 0.5, else false
func (s *Seed) RandomBoolean() bool {
	if n := s.GetByte(); n%2 == 0 {
		return true
	}
	return false
}

// GetRandomInt64 returns a random int64 generated with the seed
func (s *Seed) GetRandomInt64() int64 {
	var num int64
	for i := 0; i < 8; i++ {
		nextByte := s.GetByte()
		num |= int64(nextByte) << (i * 8)
	}
	return num
}

// GetRandomPositiveInt64 returns a random positive int64 generated with the seed
func (s *Seed) GetRandomPositiveInt64() int64 {
	return s.GetRandomInt64() & math.MaxInt
}

// GetRandomIntn returns a random int in [0, n)
func (s *Seed) GetRandomIntn(n int) int {
	return int(s.GetRandomPositiveInt64() % int64(n))
}

// RandomStringFromChoice takes in an arbitrary amount of strings and returns one of them randomly.
func (s *Seed) RandomStringFromChoice(choice ...string) string {
	return choice[s.GetRandomIntn(len(choice))]
}
