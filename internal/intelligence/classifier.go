package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// DocumentClassifier performs rule-based document classification
type DocumentClassifier struct {
	config      ClassificationConfig
	rules       []ClassificationRule
	cache       map[string]ClassificationResult
	cacheMutex  sync.RWMutex
	version     string
	logger      *log.Logger
	initialized bool
}

// NewDocumentClassifier creates a new document classifier with default configuration
func NewDocumentClassifier() *DocumentClassifier {
	return &DocumentClassifier{
		config:      DefaultClassificationConfig(),
		rules:       getDefaultRules(),
		cache:       make(map[string]ClassificationResult),
		version:     "1.0.0",
		initialized: true,
	}
}

// NewDocumentClassifierWithConfig creates a new document classifier with custom configuration
func NewDocumentClassifierWithConfig(config ClassificationConfig) *DocumentClassifier {
	classifier := &DocumentClassifier{
		config:      config,
		rules:       getDefaultRules(),
		cache:       make(map[string]ClassificationResult),
		version:     "1.0.0",
		initialized: true,
	}

	// Load custom rules if specified
	if config.EnableCustomRules && config.CustomRulesPath != "" {
		if err := classifier.LoadCustomRules(config.CustomRulesPath); err != nil {
			log.Printf("Warning: Failed to load custom rules from %s: %v", config.CustomRulesPath, err)
		}
	}

	return classifier
}

// Classify performs document classification on the given document structure
func (dc *DocumentClassifier) Classify(ctx context.Context, structure *DocumentStructure, content string) (*ClassificationResult, error) {
	if !dc.initialized {
		return nil, fmt.Errorf("classifier not initialized")
	}

	startTime := time.Now()
	analysisID := fmt.Sprintf("analysis_%d", time.Now().UnixNano())

	// Check cache first
	if dc.config.CacheClassifications {
		if cached, found := dc.getCachedResult(content); found {
			return &cached, nil
		}
	}

	// Extract document features
	features := dc.extractFeatures(structure, content)

	// Initialize classification scores
	scores := make(map[DocumentType]float64)
	reasons := make(map[DocumentType][]ClassificationReason)
	metrics := ClassificationMetrics{
		RuleScores:          make(map[string]float64),
		KeywordMatches:      make(map[string]int),
		StructureSignatures: make(map[string]bool),
		ContentHeuristics:   make(map[string]float64),
		StructuralElements:  make(map[string]int),
	}

	// Apply classification rules
	rulesApplied := []string{}
	for _, rule := range dc.rules {
		if !rule.Enabled {
			continue
		}

		// Check for context timeout
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		confidence, ruleReasons := dc.evaluateRule(rule, structure, content, features)
		if confidence >= rule.MinConfidence {
			scores[rule.DocumentType] += confidence * rule.Weight
			reasons[rule.DocumentType] = append(reasons[rule.DocumentType], ruleReasons...)
			rulesApplied = append(rulesApplied, rule.Name)
			metrics.RuleScores[rule.Name] = confidence
		}
		metrics.TotalRulesEvaluated++
	}

	// Normalize scores and determine primary classification
	primaryType, primaryConfidence := dc.determinePrimaryClassification(scores)

	// Generate alternatives
	alternatives := dc.generateAlternatives(scores, primaryType, primaryConfidence)

	// Compile final reasons
	finalReasons := reasons[primaryType]
	if len(finalReasons) == 0 {
		finalReasons = []ClassificationReason{
			{
				Rule:       "default",
				Category:   "fallback",
				Evidence:   "No strong classification signals found",
				Confidence: primaryConfidence,
				Weight:     1.0,
			},
		}
	}

	// Update metrics
	metrics.RulesMatched = len(rulesApplied)
	metrics.DocumentLength = len(content)
	metrics.PageCount = len(structure.PageStructure)
	dc.updateMetricsWithFeatures(&metrics, features)

	// Create classification result
	classification := DocumentClassification{
		Type:         primaryType,
		Confidence:   primaryConfidence,
		Alternatives: alternatives,
		Reasons:      finalReasons,
		Metrics:      metrics,
		ProcessedAt:  time.Now(),
		Version:      dc.version,
		ModelUsed:    "rule_based_v1",
	}

	result := ClassificationResult{
		Classification: classification,
		ProcessingTime: time.Since(startTime),
		RulesApplied:   rulesApplied,
		AnalysisID:     analysisID,
	}

	// Cache the result
	if dc.config.CacheClassifications {
		dc.cacheResult(content, result)
	}

	return &result, nil
}

// evaluateRule evaluates a single classification rule against the document
func (dc *DocumentClassifier) evaluateRule(rule ClassificationRule, structure *DocumentStructure, content string, features DocumentFeatures) (float64, []ClassificationReason) {
	var totalConfidence float64
	var reasons []ClassificationReason

	// Evaluate keyword rules
	if len(rule.Keywords) > 0 || len(rule.KeywordPatterns) > 0 {
		keywordConfidence, keywordReasons := dc.evaluateKeywordRules(rule, content)
		totalConfidence += keywordConfidence
		reasons = append(reasons, keywordReasons...)
	}

	// Evaluate structure rules
	if len(rule.StructureRules) > 0 {
		structureConfidence, structureReasons := dc.evaluateStructureRules(rule, structure)
		totalConfidence += structureConfidence
		reasons = append(reasons, structureReasons...)
	}

	// Evaluate content rules
	if len(rule.ContentRules) > 0 {
		contentConfidence, contentReasons := dc.evaluateContentRules(rule, content, features)
		totalConfidence += contentConfidence
		reasons = append(reasons, contentReasons...)
	}

	// Normalize confidence based on rule components
	componentCount := 0
	if len(rule.Keywords) > 0 || len(rule.KeywordPatterns) > 0 {
		componentCount++
	}
	if len(rule.StructureRules) > 0 {
		componentCount++
	}
	if len(rule.ContentRules) > 0 {
		componentCount++
	}

	if componentCount > 0 {
		totalConfidence = totalConfidence / float64(componentCount)
	}

	return totalConfidence, reasons
}

// evaluateKeywordRules evaluates keyword-based rules
func (dc *DocumentClassifier) evaluateKeywordRules(rule ClassificationRule, content string) (float64, []ClassificationReason) {
	var confidence float64
	var reasons []ClassificationReason

	contentLower := strings.ToLower(content)
	if dc.config.KeywordCaseSensitive {
		contentLower = content
	}

	// Check literal keywords
	for _, keyword := range rule.Keywords {
		searchTerm := keyword
		if !dc.config.KeywordCaseSensitive {
			searchTerm = strings.ToLower(keyword)
		}

		count := strings.Count(contentLower, searchTerm)
		if count > 0 {
			confidence += 0.1 * float64(count) // Base confidence per match
			reasons = append(reasons, ClassificationReason{
				Rule:       rule.Name,
				Category:   "keyword",
				Evidence:   fmt.Sprintf("Found keyword '%s' %d times", keyword, count),
				Confidence: 0.1 * float64(count),
				Weight:     rule.Weight,
			})
		}
	}

	// Check keyword patterns (regex)
	for _, pattern := range rule.KeywordPatterns {
		var regex *regexp.Regexp
		var err error

		if dc.config.KeywordCaseSensitive {
			regex, err = regexp.Compile(pattern)
		} else {
			regex, err = regexp.Compile("(?i)" + pattern)
		}

		if err != nil {
			continue // Skip invalid patterns
		}

		matches := regex.FindAllString(content, -1)
		if len(matches) > 0 {
			confidence += 0.15 * float64(len(matches))
			reasons = append(reasons, ClassificationReason{
				Rule:       rule.Name,
				Category:   "pattern",
				Evidence:   fmt.Sprintf("Pattern '%s' matched %d times", pattern, len(matches)),
				Confidence: 0.15 * float64(len(matches)),
				Weight:     rule.Weight,
			})
		}
	}

	return confidence, reasons
}

// evaluateStructureRules evaluates structure-based rules
func (dc *DocumentClassifier) evaluateStructureRules(rule ClassificationRule, structure *DocumentStructure) (float64, []ClassificationReason) {
	var confidence float64
	var reasons []ClassificationReason

	for _, structRule := range rule.StructureRules {
		count := dc.countStructuralElements(structure, structRule.ElementType)

		// Check if count meets requirements
		meetsMin := structRule.MinCount == 0 || count >= structRule.MinCount
		meetsMax := structRule.MaxCount == 0 || count <= structRule.MaxCount

		if meetsMin && meetsMax {
			confidence += structRule.Confidence
			reasons = append(reasons, ClassificationReason{
				Rule:       rule.Name,
				Category:   "structure",
				Evidence:   fmt.Sprintf("Found %d '%s' elements (expected %d-%d)", count, structRule.ElementType, structRule.MinCount, structRule.MaxCount),
				Confidence: structRule.Confidence,
				Weight:     rule.Weight,
			})
		}
	}

	return confidence, reasons
}

// evaluateContentRules evaluates content-based rules
func (dc *DocumentClassifier) evaluateContentRules(rule ClassificationRule, content string, features DocumentFeatures) (float64, []ClassificationReason) {
	var confidence float64
	var reasons []ClassificationReason

	for _, contentRule := range rule.ContentRules {
		matches := dc.evaluateContentPattern(contentRule, content)

		if matches >= contentRule.MinMatches && (contentRule.MaxMatches == 0 || matches <= contentRule.MaxMatches) {
			confidence += contentRule.Confidence
			reasons = append(reasons, ClassificationReason{
				Rule:       rule.Name,
				Category:   "content",
				Evidence:   fmt.Sprintf("Content rule '%s' matched %d times", contentRule.RuleType, matches),
				Confidence: contentRule.Confidence,
				Weight:     rule.Weight,
			})
		}
	}

	return confidence, reasons
}

// evaluateContentPattern evaluates a content pattern
func (dc *DocumentClassifier) evaluateContentPattern(rule ContentRule, content string) int {
	switch rule.RuleType {
	case "regex":
		var regex *regexp.Regexp
		var err error

		if rule.CaseSensitive {
			regex, err = regexp.Compile(rule.Pattern)
		} else {
			regex, err = regexp.Compile("(?i)" + rule.Pattern)
		}

		if err != nil {
			return 0
		}

		matches := regex.FindAllString(content, -1)
		return len(matches)

	case "contains":
		searchContent := content
		searchPattern := rule.Pattern

		if !rule.CaseSensitive {
			searchContent = strings.ToLower(content)
			searchPattern = strings.ToLower(rule.Pattern)
		}

		return strings.Count(searchContent, searchPattern)

	case "starts_with":
		searchContent := content
		searchPattern := rule.Pattern

		if !rule.CaseSensitive {
			searchContent = strings.ToLower(content)
			searchPattern = strings.ToLower(rule.Pattern)
		}

		if strings.HasPrefix(searchContent, searchPattern) {
			return 1
		}
		return 0

	case "ends_with":
		searchContent := content
		searchPattern := rule.Pattern

		if !rule.CaseSensitive {
			searchContent = strings.ToLower(content)
			searchPattern = strings.ToLower(rule.Pattern)
		}

		if strings.HasSuffix(searchContent, searchPattern) {
			return 1
		}
		return 0

	default:
		return 0
	}
}

// countStructuralElements counts elements of a specific type in the document structure
func (dc *DocumentClassifier) countStructuralElements(structure *DocumentStructure, elementType string) int {
	count := 0

	// Convert to lowercase for comparison
	elementType = strings.ToLower(elementType)

	// Walk through the structure tree
	dc.walkStructureTree(structure.Root, func(node *StructureNode) {
		if strings.ToLower(string(node.Type)) == elementType {
			count++
		}
	})

	return count
}

// walkStructureTree walks through the structure tree and applies a function to each node
func (dc *DocumentClassifier) walkStructureTree(node *StructureNode, fn func(*StructureNode)) {
	if node == nil {
		return
	}

	fn(node)

	for _, child := range node.Children {
		dc.walkStructureTree(child, fn)
	}
}

// extractFeatures extracts document features for classification
func (dc *DocumentClassifier) extractFeatures(structure *DocumentStructure, content string) DocumentFeatures {
	features := DocumentFeatures{
		PageCount:       len(structure.PageStructure),
		WordCount:       len(strings.Fields(content)),
		ParagraphCount:  strings.Count(content, "\n\n") + 1,
		NumericPatterns: make(map[string]int),
	}

	// Count structural elements
	dc.walkStructureTree(structure.Root, func(node *StructureNode) {
		switch node.Type {
		case StructureTypeHeader:
			features.HeaderCount++
		case StructureTypeList:
			features.ListCount++
		case StructureTypeTable:
			features.TableCount++
		case StructureTypeImage:
			features.ImageCount++
		}
	})

	// Extract header levels
	features.HeaderLevels = make([]int, 0)
	dc.walkStructureTree(structure.Root, func(node *StructureNode) {
		if node.Type == StructureTypeHeader {
			features.HeaderLevels = append(features.HeaderLevels, node.Level)
		}
	})

	// Count unique words
	words := strings.Fields(strings.ToLower(content))
	uniqueWords := make(map[string]bool)
	totalLength := 0
	for _, word := range words {
		// Clean word of punctuation
		word = strings.TrimFunc(word, func(c rune) bool {
			return !unicode.IsLetter(c) && !unicode.IsNumber(c)
		})
		if len(word) > 0 {
			uniqueWords[word] = true
			totalLength += len(word)
		}
	}
	features.UniqueWords = len(uniqueWords)
	if features.WordCount > 0 {
		features.AverageWordLength = float64(totalLength) / float64(features.WordCount)
	}

	// Count sentences (rough estimate)
	features.SentenceCount = strings.Count(content, ".") + strings.Count(content, "!") + strings.Count(content, "?")

	// Count various patterns
	features.EmailCount = len(regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`).FindAllString(content, -1))
	features.URLCount = len(regexp.MustCompile(`https?://[^\s]+`).FindAllString(content, -1))

	// Numeric patterns
	features.NumericPatterns["currency"] = len(regexp.MustCompile(`\$[\d,]+\.?\d*|\d+\.\d{2}\s*(USD|EUR|GBP)`).FindAllString(content, -1))
	features.NumericPatterns["date"] = len(regexp.MustCompile(`\d{1,2}[/-]\d{1,2}[/-]\d{2,4}|\d{4}-\d{2}-\d{2}`).FindAllString(content, -1))
	features.NumericPatterns["phone"] = len(regexp.MustCompile(`\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`).FindAllString(content, -1))

	// Quality metrics
	if len(content) > 0 {
		textLength := float64(len(content))
		nonWhitespace := float64(len(strings.ReplaceAll(content, " ", "")))
		features.WhitespaceRatio = (textLength - nonWhitespace) / textLength
		features.TextDensity = nonWhitespace / textLength
	}

	return features
}

// determinePrimaryClassification determines the primary classification from scores
func (dc *DocumentClassifier) determinePrimaryClassification(scores map[DocumentType]float64) (DocumentType, float64) {
	if len(scores) == 0 {
		return DocumentTypeUnknown, 0.0
	}

	var maxType DocumentType
	var maxScore float64

	for docType, score := range scores {
		if score > maxScore {
			maxScore = score
			maxType = docType
		}
	}

	// Normalize confidence to 0-1 range
	if maxScore > 1.0 {
		maxScore = 1.0
	}

	// Check if confidence meets threshold
	if maxScore < dc.config.MinConfidenceThreshold {
		return DocumentTypeUnknown, maxScore
	}

	return maxType, maxScore
}

// generateAlternatives generates alternative classifications
func (dc *DocumentClassifier) generateAlternatives(scores map[DocumentType]float64, primaryType DocumentType, primaryConfidence float64) []ClassificationAlternative {
	var alternatives []ClassificationAlternative

	// Sort scores by confidence
	type scoreEntry struct {
		docType    DocumentType
		confidence float64
	}

	var sortedScores []scoreEntry
	for docType, score := range scores {
		if docType != primaryType && score >= dc.config.MinConfidenceThreshold*0.5 {
			sortedScores = append(sortedScores, scoreEntry{docType, score})
		}
	}

	sort.Slice(sortedScores, func(i, j int) bool {
		return sortedScores[i].confidence > sortedScores[j].confidence
	})

	// Take top alternatives
	maxAlternatives := dc.config.MaxAlternatives
	if maxAlternatives > len(sortedScores) {
		maxAlternatives = len(sortedScores)
	}

	for i := 0; i < maxAlternatives; i++ {
		entry := sortedScores[i]
		alternatives = append(alternatives, ClassificationAlternative{
			Type:       entry.docType,
			Confidence: entry.confidence,
			Reasons:    []string{fmt.Sprintf("Score: %.2f", entry.confidence)},
		})
	}

	return alternatives
}

// updateMetricsWithFeatures updates metrics with extracted features
func (dc *DocumentClassifier) updateMetricsWithFeatures(metrics *ClassificationMetrics, features DocumentFeatures) {
	metrics.DocumentLength = features.WordCount
	metrics.PageCount = features.PageCount

	// Calculate quality metrics
	metrics.TextQuality = dc.calculateTextQuality(features)
	metrics.StructureQuality = dc.calculateStructureQuality(features)
	metrics.OverallQuality = (metrics.TextQuality + metrics.StructureQuality) / 2.0
}

// calculateTextQuality calculates text quality score
func (dc *DocumentClassifier) calculateTextQuality(features DocumentFeatures) float64 {
	score := 0.0

	// Word count quality
	if features.WordCount > 100 {
		score += 0.3
	} else if features.WordCount > 50 {
		score += 0.2
	} else if features.WordCount > 10 {
		score += 0.1
	}

	// Vocabulary diversity
	if features.WordCount > 0 {
		diversity := float64(features.UniqueWords) / float64(features.WordCount)
		score += diversity * 0.3
	}

	// Average word length (reasonable range)
	if features.AverageWordLength >= 4.0 && features.AverageWordLength <= 8.0 {
		score += 0.2
	}

	// Sentence structure
	if features.SentenceCount > 0 && features.WordCount > 0 {
		avgWordsPerSentence := float64(features.WordCount) / float64(features.SentenceCount)
		if avgWordsPerSentence >= 10.0 && avgWordsPerSentence <= 25.0 {
			score += 0.2
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateStructureQuality calculates structure quality score
func (dc *DocumentClassifier) calculateStructureQuality(features DocumentFeatures) float64 {
	score := 0.0

	// Has headers
	if features.HeaderCount > 0 {
		score += 0.3
	}

	// Has proper structure
	totalStructuralElements := features.ListCount + features.TableCount + features.HeaderCount
	if totalStructuralElements > 0 {
		score += 0.3
	}

	// Balanced content
	if features.ParagraphCount > 0 {
		score += 0.2
	}

	// Has multimedia elements
	if features.ImageCount > 0 {
		score += 0.2
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// Cache management methods

func (dc *DocumentClassifier) getCachedResult(content string) (ClassificationResult, bool) {
	dc.cacheMutex.RLock()
	defer dc.cacheMutex.RUnlock()

	key := dc.generateCacheKey(content)
	result, found := dc.cache[key]
	return result, found
}

func (dc *DocumentClassifier) cacheResult(content string, result ClassificationResult) {
	dc.cacheMutex.Lock()
	defer dc.cacheMutex.Unlock()

	key := dc.generateCacheKey(content)
	dc.cache[key] = result

	// Simple cache size management
	if len(dc.cache) > 100 {
		// Remove oldest entries (simple approach)
		for k := range dc.cache {
			delete(dc.cache, k)
			break
		}
	}
}

func (dc *DocumentClassifier) generateCacheKey(content string) string {
	// Simple hash of content length and first/last characters
	if len(content) == 0 {
		return "empty"
	}

	first := string(content[0])
	last := string(content[len(content)-1])
	return fmt.Sprintf("%s_%d_%s", first, len(content), last)
}

// LoadCustomRules loads custom classification rules from a file
func (dc *DocumentClassifier) LoadCustomRules(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read custom rules file: %w", err)
	}

	var ruleSet ClassificationRuleSet
	if err := json.Unmarshal(data, &ruleSet); err != nil {
		return fmt.Errorf("failed to parse custom rules: %w", err)
	}

	// Append custom rules to existing rules
	dc.rules = append(dc.rules, ruleSet.Rules...)

	return nil
}

// GetVersion returns the classifier version
func (dc *DocumentClassifier) GetVersion() string {
	return dc.version
}

// GetConfig returns the current configuration
func (dc *DocumentClassifier) GetConfig() ClassificationConfig {
	return dc.config
}

// SetConfig updates the classifier configuration
func (dc *DocumentClassifier) SetConfig(config ClassificationConfig) {
	dc.config = config
}
