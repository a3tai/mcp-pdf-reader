package intelligence

// getDefaultRules returns the default set of classification rules
func getDefaultRules() []ClassificationRule {
	return []ClassificationRule{
		// Invoice Rules
		{
			Name:         "invoice_keywords",
			DocumentType: DocumentTypeInvoice,
			Category:     "keyword",
			Keywords: []string{
				"invoice", "bill", "billing", "payment", "due", "amount", "total",
				"subtotal", "tax", "vat", "discount", "item", "quantity", "price",
				"invoice number", "invoice date", "bill to", "ship to", "vendor",
				"customer", "remit", "payment terms", "net 30", "due date",
			},
			KeywordPatterns: []string{
				`(?i)invoice\s*#?\s*\d+`,
				`(?i)bill\s*#?\s*\d+`,
				`(?i)\$[\d,]+\.?\d*`,
				`(?i)total\s*:?\s*\$[\d,]+\.?\d*`,
				`(?i)amount\s*due\s*:?\s*\$[\d,]+\.?\d*`,
			},
			Weight:        0.8,
			MinConfidence: 0.3,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies invoices based on common invoice keywords and patterns",
		},
		{
			Name:         "invoice_structure",
			DocumentType: DocumentTypeInvoice,
			Category:     "structure",
			StructureRules: []StructureRule{
				{
					ElementType: "table",
					MinCount:    1,
					MaxCount:    0,
					Confidence:  0.4,
				},
				{
					ElementType: "header",
					MinCount:    1,
					MaxCount:    0,
					Confidence:  0.2,
				},
			},
			Weight:        0.6,
			MinConfidence: 0.2,
			Priority:      2,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies invoices based on typical structural elements",
		},

		// Report Rules
		{
			Name:         "report_keywords",
			DocumentType: DocumentTypeReport,
			Category:     "keyword",
			Keywords: []string{
				"report", "analysis", "summary", "findings", "conclusion",
				"executive summary", "overview", "background", "methodology",
				"results", "recommendations", "appendix", "references",
				"quarterly", "annual", "monthly", "status report",
			},
			KeywordPatterns: []string{
				`(?i)(annual|quarterly|monthly|weekly)\s+report`,
				`(?i)executive\s+summary`,
				`(?i)table\s+of\s+contents`,
				`(?i)(section|chapter)\s+\d+`,
			},
			Weight:        0.7,
			MinConfidence: 0.3,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies reports based on common report terminology",
		},
		{
			Name:         "report_structure",
			DocumentType: DocumentTypeReport,
			Category:     "structure",
			StructureRules: []StructureRule{
				{
					ElementType: "header",
					MinCount:    3,
					MaxCount:    0,
					Confidence:  0.4,
				},
				{
					ElementType: "table",
					MinCount:    1,
					MaxCount:    0,
					Confidence:  0.3,
				},
			},
			Weight:        0.6,
			MinConfidence: 0.2,
			Priority:      2,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies reports based on hierarchical structure",
		},

		// Form Rules
		{
			Name:         "form_keywords",
			DocumentType: DocumentTypeForm,
			Category:     "keyword",
			Keywords: []string{
				"form", "application", "questionnaire", "survey", "checkbox",
				"please fill", "complete", "signature", "date", "name",
				"address", "phone", "email", "submit", "required field",
				"optional", "instructions", "please print", "check one",
			},
			KeywordPatterns: []string{
				`(?i)\[\s*\]|\[x\]|\[\s*x\s*\]`, // checkboxes
				`(?i)signature\s*:?\s*_+`,       // signature lines
				`(?i)date\s*:?\s*_+`,            // date lines
				`(?i)name\s*:?\s*_+`,            // name lines
			},
			Weight:        0.8,
			MinConfidence: 0.4,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies forms based on form-specific keywords and patterns",
		},

		// Contract Rules
		{
			Name:         "contract_keywords",
			DocumentType: DocumentTypeContract,
			Category:     "keyword",
			Keywords: []string{
				"contract", "agreement", "terms", "conditions", "parties",
				"whereas", "party", "covenant", "obligation", "breach",
				"liability", "indemnity", "termination", "effective date",
				"term", "renewal", "governing law", "jurisdiction",
				"arbitration", "dispute", "remedy", "damages",
			},
			KeywordPatterns: []string{
				`(?i)this\s+agreement`,
				`(?i)whereas\s+.+`,
				`(?i)party\s+of\s+the\s+(first|second)\s+part`,
				`(?i)in\s+witness\s+whereof`,
				`(?i)executed\s+on\s+.+`,
			},
			Weight:        0.9,
			MinConfidence: 0.4,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies contracts based on legal terminology",
		},

		// Academic Paper Rules
		{
			Name:         "academic_keywords",
			DocumentType: DocumentTypeAcademic,
			Category:     "keyword",
			Keywords: []string{
				"abstract", "introduction", "methodology", "literature review",
				"hypothesis", "research", "study", "analysis", "results",
				"discussion", "conclusion", "references", "bibliography",
				"citation", "peer review", "journal", "proceedings",
				"university", "department", "professor", "phd", "doctoral",
			},
			KeywordPatterns: []string{
				`(?i)doi\s*:\s*[\d\.\/]+`,
				`(?i)\d{4}\)\s*.+\.\s*journal`,
				`(?i)et\s+al\.`,
				`(?i)vol\.\s*\d+`,
				`(?i)pp\.\s*\d+-\d+`,
			},
			Weight:        0.8,
			MinConfidence: 0.4,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies academic papers based on scholarly terminology",
		},

		// Manual/Guide Rules
		{
			Name:         "manual_keywords",
			DocumentType: DocumentTypeManual,
			Category:     "keyword",
			Keywords: []string{
				"manual", "guide", "instructions", "handbook", "tutorial",
				"how to", "step", "procedure", "process", "warning",
				"caution", "note", "tip", "important", "installation",
				"setup", "configuration", "troubleshooting", "faq",
				"user guide", "reference", "index", "glossary",
			},
			KeywordPatterns: []string{
				`(?i)step\s+\d+`,
				`(?i)chapter\s+\d+`,
				`(?i)section\s+\d+\.\d+`,
				`(?i)figure\s+\d+`,
				`(?i)table\s+\d+`,
			},
			Weight:        0.7,
			MinConfidence: 0.3,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies manuals and guides based on instructional content",
		},

		// Letter Rules
		{
			Name:         "letter_keywords",
			DocumentType: DocumentTypeLetter,
			Category:     "keyword",
			Keywords: []string{
				"dear", "sincerely", "regards", "yours truly", "best regards",
				"cordially", "respectfully", "faithfully", "letter",
				"correspondence", "memo", "memorandum", "attention",
				"subject", "re:", "cc:", "bcc:", "enclosure", "attachment",
			},
			KeywordPatterns: []string{
				`(?i)dear\s+[a-z\s]+,`,
				`(?i)sincerely\s*,?\s*yours`,
				`(?i)best\s+regards\s*,`,
				`(?i)yours\s+(truly|faithfully)\s*,`,
				`(?i)cc\s*:\s*.+`,
			},
			Weight:        0.8,
			MinConfidence: 0.4,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies letters based on common letter formatting",
		},

		// Brochure Rules
		{
			Name:         "brochure_keywords",
			DocumentType: DocumentTypeBrochure,
			Category:     "keyword",
			Keywords: []string{
				"brochure", "flyer", "pamphlet", "marketing", "promotion",
				"special offer", "limited time", "call now", "visit us",
				"contact us", "learn more", "discover", "experience",
				"featured", "benefits", "advantages", "testimonials",
				"satisfaction guaranteed", "free trial", "discount",
			},
			KeywordPatterns: []string{
				`(?i)call\s+\d{3}[-\.\s]?\d{3}[-\.\s]?\d{4}`,
				`(?i)visit\s+us\s+at`,
				`(?i)\d+%\s+off`,
				`(?i)limited\s+time\s+offer`,
				`(?i)satisfaction\s+guaranteed`,
			},
			Weight:        0.7,
			MinConfidence: 0.3,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies brochures and marketing materials",
		},

		// Technical Document Rules
		{
			Name:         "technical_keywords",
			DocumentType: DocumentTypeTechnical,
			Category:     "keyword",
			Keywords: []string{
				"technical", "specification", "api", "documentation",
				"function", "parameter", "return", "exception", "class",
				"method", "variable", "algorithm", "implementation",
				"architecture", "design", "system", "component",
				"interface", "protocol", "framework", "library",
			},
			KeywordPatterns: []string{
				`(?i)function\s+\w+\s*\(`,
				`(?i)class\s+\w+`,
				`(?i)def\s+\w+\s*\(`,
				`(?i)public\s+class`,
				`(?i)#include\s*<.+>`,
			},
			Weight:        0.8,
			MinConfidence: 0.4,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies technical documents based on programming and technical terminology",
		},

		// Financial Statement Rules
		{
			Name:         "financial_keywords",
			DocumentType: DocumentTypeFinancial,
			Category:     "keyword",
			Keywords: []string{
				"balance sheet", "income statement", "cash flow", "profit",
				"loss", "revenue", "expense", "asset", "liability", "equity",
				"financial", "accounting", "audit", "fiscal", "quarter",
				"year ended", "consolidated", "statement", "gaap",
				"depreciation", "amortization", "earnings", "dividend",
			},
			KeywordPatterns: []string{
				`(?i)year\s+ended\s+\w+\s+\d+`,
				`(?i)quarter\s+ended\s+\w+\s+\d+`,
				`(?i)\$\d+,?\d*\s*(million|billion|thousand)`,
				`(?i)net\s+(income|loss)\s*\$`,
				`(?i)total\s+(assets|liabilities|equity)\s*\$`,
			},
			Weight:        0.9,
			MinConfidence: 0.4,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies financial statements and reports",
		},

		// Legal Document Rules
		{
			Name:         "legal_keywords",
			DocumentType: DocumentTypeLegal,
			Category:     "keyword",
			Keywords: []string{
				"legal", "court", "judge", "attorney", "lawyer", "counsel",
				"plaintiff", "defendant", "case", "lawsuit", "litigation",
				"motion", "order", "ruling", "judgment", "verdict",
				"statute", "regulation", "law", "legal brief", "petition",
				"affidavit", "deposition", "subpoena", "evidence",
			},
			KeywordPatterns: []string{
				`(?i)case\s+no\.\s*\d+`,
				`(?i)v\.\s+\w+`, // versus in case names
				`(?i)honorable\s+\w+`,
				`(?i)court\s+of\s+\w+`,
				`(?i)state\s+of\s+\w+\s+v\.\s+\w+`,
			},
			Weight:        0.9,
			MinConfidence: 0.4,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies legal documents based on legal terminology",
		},

		// Resume Rules
		{
			Name:         "resume_keywords",
			DocumentType: DocumentTypeResume,
			Category:     "keyword",
			Keywords: []string{
				"resume", "curriculum vitae", "cv", "experience", "education",
				"skills", "objective", "summary", "employment", "work history",
				"achievements", "accomplishments", "qualifications",
				"certifications", "references", "contact information",
				"professional", "career", "position", "responsibilities",
			},
			KeywordPatterns: []string{
				`(?i)phone\s*:\s*\d{3}[-\.\s]?\d{3}[-\.\s]?\d{4}`,
				`(?i)email\s*:\s*\w+@\w+\.\w+`,
				`(?i)\d{4}\s*-\s*\d{4}`, // date ranges
				`(?i)\d{4}\s*-\s*present`,
				`(?i)bachelor|master|phd|degree`,
			},
			Weight:        0.8,
			MinConfidence: 0.4,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies resumes and CVs based on typical resume content",
		},

		// Presentation Rules
		{
			Name:         "presentation_keywords",
			DocumentType: DocumentTypePresentation,
			Category:     "keyword",
			Keywords: []string{
				"presentation", "slide", "agenda", "overview", "outline",
				"thank you", "questions", "discussion", "next steps",
				"key points", "summary", "takeaways", "objectives",
				"goals", "timeline", "roadmap", "strategy", "vision",
				"mission", "proposal", "recommendation", "conclusion",
			},
			KeywordPatterns: []string{
				`(?i)slide\s+\d+`,
				`(?i)agenda\s*:`,
				`(?i)key\s+points\s*:`,
				`(?i)next\s+steps\s*:`,
				`(?i)thank\s+you\s+for\s+your\s+attention`,
			},
			Weight:        0.7,
			MinConfidence: 0.3,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies presentation slides based on presentation terminology",
		},

		// Newsletter Rules
		{
			Name:         "newsletter_keywords",
			DocumentType: DocumentTypeNewsletter,
			Category:     "keyword",
			Keywords: []string{
				"newsletter", "issue", "volume", "edition", "news",
				"updates", "announcements", "events", "upcoming",
				"featured", "spotlight", "member", "community",
				"subscribe", "unsubscribe", "forward", "share",
				"monthly", "weekly", "quarterly", "editor", "publisher",
			},
			KeywordPatterns: []string{
				`(?i)volume\s+\d+`,
				`(?i)issue\s+\d+`,
				`(?i)edition\s+\d+`,
				`(?i)\w+\s+\d{1,2},\s+\d{4}`, // date format
				`(?i)in\s+this\s+issue`,
			},
			Weight:        0.7,
			MinConfidence: 0.3,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies newsletters based on newsletter-specific terminology",
		},

		// Catalog Rules
		{
			Name:         "catalog_keywords",
			DocumentType: DocumentTypeCatalog,
			Category:     "keyword",
			Keywords: []string{
				"catalog", "catalogue", "product", "item", "price",
				"order", "model", "description", "features", "specifications",
				"available", "stock", "inventory", "part number", "sku",
				"quantity", "discount", "wholesale", "retail", "sale",
				"new arrivals", "bestsellers", "featured products",
			},
			KeywordPatterns: []string{
				`(?i)item\s*#?\s*\d+`,
				`(?i)model\s*#?\s*\w+`,
				`(?i)price\s*:?\s*\$[\d,]+\.?\d*`,
				`(?i)part\s*#?\s*\w+`,
				`(?i)sku\s*:?\s*\w+`,
			},
			Weight:        0.7,
			MinConfidence: 0.3,
			Priority:      1,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Identifies product catalogs based on commerce terminology",
		},

		// Content-based rules for better accuracy
		{
			Name:         "invoice_content_patterns",
			DocumentType: DocumentTypeInvoice,
			Category:     "content",
			ContentRules: []ContentRule{
				{
					RuleType:      "regex",
					Pattern:       `(?i)invoice\s*(?:number|#|no\.?)\s*:?\s*(\w+)`,
					CaseSensitive: false,
					MinMatches:    1,
					MaxMatches:    0,
					Confidence:    0.6,
				},
				{
					RuleType:      "regex",
					Pattern:       `(?i)total\s*(?:amount|due)?\s*:?\s*\$[\d,]+\.?\d*`,
					CaseSensitive: false,
					MinMatches:    1,
					MaxMatches:    0,
					Confidence:    0.5,
				},
			},
			Weight:        0.8,
			MinConfidence: 0.3,
			Priority:      2,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Advanced pattern matching for invoice identification",
		},

		{
			Name:         "form_content_patterns",
			DocumentType: DocumentTypeForm,
			Category:     "content",
			ContentRules: []ContentRule{
				{
					RuleType:      "regex",
					Pattern:       `\[\s*\]|\[x\]|\[\s*x\s*\]`, // checkbox patterns
					CaseSensitive: false,
					MinMatches:    3,
					MaxMatches:    0,
					Confidence:    0.7,
				},
				{
					RuleType:      "regex",
					Pattern:       `_{3,}`, // underlines for filling
					CaseSensitive: false,
					MinMatches:    5,
					MaxMatches:    0,
					Confidence:    0.5,
				},
			},
			Weight:        0.9,
			MinConfidence: 0.4,
			Priority:      2,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Advanced pattern matching for form identification",
		},
	}
}

// getDefaultStructureRules returns structure-based classification rules
func getDefaultStructureRules() []ClassificationRule {
	return []ClassificationRule{
		{
			Name:         "table_heavy_documents",
			DocumentType: DocumentTypeReport,
			Category:     "structure",
			StructureRules: []StructureRule{
				{
					ElementType: "table",
					MinCount:    3,
					MaxCount:    0,
					Confidence:  0.4,
				},
			},
			Weight:        0.6,
			MinConfidence: 0.2,
			Priority:      3,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Documents with many tables are likely reports",
		},
		{
			Name:         "hierarchical_structure",
			DocumentType: DocumentTypeManual,
			Category:     "structure",
			StructureRules: []StructureRule{
				{
					ElementType: "header",
					MinCount:    5,
					MaxCount:    0,
					Confidence:  0.3,
				},
				{
					ElementType: "list",
					MinCount:    2,
					MaxCount:    0,
					Confidence:  0.2,
				},
			},
			Weight:        0.5,
			MinConfidence: 0.2,
			Priority:      3,
			Enabled:       true,
			Version:       "1.0",
			Description:   "Documents with hierarchical structure are likely manuals",
		},
	}
}

// GetAllDefaultRules returns all default classification rules
func GetAllDefaultRules() []ClassificationRule {
	rules := getDefaultRules()
	rules = append(rules, getDefaultStructureRules()...)
	return rules
}
