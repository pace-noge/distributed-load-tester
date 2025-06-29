package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// Lists of words for generating memorable worker names
var (
	adjectives = []string{
		"Swift", "Mighty", "Blazing", "Thunder", "Lightning", "Turbo", "Rocket", "Phantom",
		"Stealth", "Cyber", "Neon", "Quantum", "Atomic", "Cosmic", "Solar", "Lunar",
		"Storm", "Fire", "Ice", "Wind", "Shadow", "Golden", "Silver", "Diamond",
		"Elite", "Prime", "Alpha", "Beta", "Gamma", "Delta", "Omega", "Ultra",
		"Super", "Hyper", "Mega", "Giga", "Tera", "Nova", "Stellar", "Galactic",
	}

	nouns = []string{
		"Falcon", "Eagle", "Hawk", "Phoenix", "Dragon", "Tiger", "Lion", "Wolf",
		"Panther", "Cheetah", "Viper", "Cobra", "Shark", "Whale", "Dolphin", "Orca",
		"Knight", "Warrior", "Guardian", "Sentinel", "Hunter", "Ranger", "Scout", "Ninja",
		"Samurai", "Gladiator", "Champion", "Hero", "Legend", "Master", "Ace", "Chief",
		"Commander", "Captain", "Admiral", "General", "Marshal", "Baron", "Duke", "King",
		"Striker", "Crusher", "Breaker", "Destroyer", "Annihilator", "Terminator", "Obliterator", "Devastator",
	}

	colors = []string{
		"Red", "Blue", "Green", "Purple", "Orange", "Yellow", "Pink", "Cyan",
		"Magenta", "Violet", "Crimson", "Scarlet", "Azure", "Emerald", "Amber", "Coral",
		"Indigo", "Turquoise", "Lime", "Teal", "Navy", "Maroon", "Olive", "Aqua",
	}
)

// GenerateWorkerName creates a unique, memorable worker name
// Format: {Adjective}{Color}{Noun}-{UniqueID}
// Examples: SwiftRedFalcon-7X2K, MightyBluePhoenix-9M4L
func GenerateWorkerName() string {
	rand.Seed(time.Now().UnixNano())

	adjective := adjectives[rand.Intn(len(adjectives))]
	color := colors[rand.Intn(len(colors))]
	noun := nouns[rand.Intn(len(nouns))]

	// Generate a unique suffix with numbers and letters
	suffix := generateUniqueSuffix()

	return fmt.Sprintf("%s%s%s-%s", adjective, color, noun, suffix)
}

// GenerateTestName creates a memorable test name
// Format: {Adjective}-{Noun}-Test-{Timestamp}
// Examples: Lightning-Strike-Test-20250630, Quantum-Phoenix-Test-20250630
func GenerateTestName() string {
	rand.Seed(time.Now().UnixNano())

	adjective := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	timestamp := time.Now().Format("20060102")

	return fmt.Sprintf("%s-%s-Test-%s", adjective, noun, timestamp)
}

// generateUniqueSuffix creates a short unique identifier
func generateUniqueSuffix() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())

	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = charset[rand.Intn(len(charset))]
	}

	return string(suffix)
}

// ValidateWorkerName checks if a worker name follows the expected format
func ValidateWorkerName(name string) bool {
	// Basic validation - should contain at least one hyphen and be reasonable length
	return len(name) >= 10 && len(name) <= 50 && containsHyphen(name)
}

func containsHyphen(s string) bool {
	for _, char := range s {
		if char == '-' {
			return true
		}
	}
	return false
}

// GetWorkerDisplayName extracts a display-friendly version of the worker name
// SwiftRedFalcon-7X2K -> Swift Red Falcon
func GetWorkerDisplayName(workerName string) string {
	// Find the hyphen and take everything before it
	for i, char := range workerName {
		if char == '-' {
			fullName := workerName[:i]
			// Insert spaces before capital letters (except the first one)
			var result []rune
			for j, r := range fullName {
				if j > 0 && r >= 'A' && r <= 'Z' {
					result = append(result, ' ')
				}
				result = append(result, r)
			}
			return string(result)
		}
	}
	return workerName
}
