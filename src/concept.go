package main

type conceptDictionary struct {
	conceptsMap map[string]*concept
}

type concept struct {
	conceptStep *step
	fileName    string
}

type conceptParser struct {
	currentState   int
	currentConcept *step
}

func (parser *conceptParser) parse(text string) ([]*step, *parseError) {
	defer parser.resetState()

	specParser := new(specParser)
	tokens, err := specParser.generateTokens(text)
	if err != nil {
		return nil, err
	}
	return parser.createConcepts(tokens)
}

func (parser *conceptParser) resetState() {
	parser.currentState = 0
	parser.currentConcept = nil
}

func (parser *conceptParser) createConcepts(tokens []*token) ([]*step, *parseError) {
	concepts := make([]*step, 0)
	var error *parseError
	for _, token := range tokens {
		if parser.isConceptHeading(token) {
			if isInState(parser.currentState, conceptScope, stepScope) {
				concepts = append(concepts, parser.currentConcept)
			}
			parser.currentConcept, error = parser.processConceptHeading(token)
			if error != nil {
				return nil, error
			}
			addStates(&parser.currentState, conceptScope)
		} else if parser.isStep(token) {
			if !isInState(parser.currentState, conceptScope) {
				return nil, &parseError{lineNo: token.lineNo, message: "Step is not defined inside a concept heading", lineText: token.lineText}
			}
			if err := parser.processConceptStep(token); err != nil {
				return nil, err
			}
			addStates(&parser.currentState, stepScope)
		} else if parser.isTableHeader(token) {
			if !isInState(parser.currentState, stepScope) {
				return nil, &parseError{lineNo: token.lineNo, message: "Table doesn't belong to any step", lineText: token.lineText}
			}
			if err := parser.processTableHeader(token); err != nil {
				return nil, err
			}
			addStates(&parser.currentState, tableScope)
		} else if parser.isTableDataRow(token) {
			if err := parser.processTableDataRow(token); err != nil {
				return nil, err
			}
		}
	}
	if !isInState(parser.currentState, stepScope) {
		return nil, &parseError{lineNo: parser.currentConcept.lineNo, message: "Concept should have atleast one step", lineText: parser.currentConcept.lineText}
	}
	return append(concepts, parser.currentConcept), nil
}

func (parser *conceptParser) isConceptHeading(token *token) bool {
	return token.kind == specKind || token.kind == scenarioKind
}

func (parser *conceptParser) isStep(token *token) bool {
	return token.kind == stepKind
}

func (parser *conceptParser) isTableHeader(token *token) bool {
	return token.kind == tableHeader
}

func (parser *conceptParser) isTableDataRow(token *token) bool {
	return token.kind == tableRow
}

func (parser *conceptParser) processConceptHeading(token *token) (*step, *parseError) {
	processStep(new(specParser), token)
	concept, err := new(specification).createStep(token, nil)
	if err != nil {
		return nil, err
	}
	concept.isConcept = true
	if !parser.hasOnlyDynamicParams(concept) {
		return nil, &parseError{lineNo: concept.lineNo, message: "Concept heading can have only Dynamic Parameters"}
	}
	parser.createConceptLookup(concept)
	return concept, nil

}

func (parser *conceptParser) processConceptStep(token *token) *parseError {
	processStep(new(specParser), token)
	conceptStep, err := new(specification).createStep(token, &parser.currentConcept.lookup)
	if err != nil {
		return err
	}
	parser.currentConcept.conceptSteps = append(parser.currentConcept.conceptSteps, conceptStep)
	return nil
}

func (parser *conceptParser) processTableHeader(token *token) *parseError {
	steps := parser.currentConcept.conceptSteps
	currentStep := steps[len(steps)-1]
	currentStep.inlineTable.addHeaders(token.args)
	return nil
}

func (parser *conceptParser) processTableDataRow(token *token) *parseError {
	steps := parser.currentConcept.conceptSteps
	currentStep := steps[len(steps)-1]
	currentStep.inlineTable.addRowValues(token.args)
	return nil
}

func (parser *conceptParser) hasOnlyDynamicParams(concept *step) bool {
	for _, arg := range concept.args {
		if arg.argType != dynamic {
			return false
		}
	}
	return true
}

func (parser *conceptParser) createConceptLookup(concept *step) {
	for _, arg := range concept.args {
		concept.lookup.addArgName(arg.value)
	}
}

func (conceptDictionary *conceptDictionary) add(concepts []*step, conceptFile string) {
	if conceptDictionary.conceptsMap == nil {
		conceptDictionary.conceptsMap = make(map[string]*concept)
	}
	for _, step := range concepts {
		conceptDictionary.conceptsMap[step.value] = &concept{step, conceptFile}
	}
}

func (conceptDictionary *conceptDictionary) search(stepValue string) *concept {
	if concept, ok := conceptDictionary.conceptsMap[stepValue]; ok {
		return concept
	}
	return nil
}
