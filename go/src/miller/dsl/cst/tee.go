// ================================================================
// This handles tee statements.
// ================================================================

package cst

import (
	"errors"
	"fmt"
	"os"

	"miller/dsl"
	"miller/lib"
	"miller/types"
)

// ----------------------------------------------------------------
// Examples:
//   tee > "foo.dat", $*
//   tee > stderr, $*
//   tee > stdout, $*
//   tee | "jq .", $*
//
// The item being teed can only be $*. This is a special case of emit.  (This
// doesn't do anything emit can't do.)
//
// $ mlr -n put -v 'tee > stdout, $*'
// DSL EXPRESSION:
// tee > stdout, $*
// RAW AST:
// * statement block
//     * tee statement "tee"
//         * full record "$*"
//         * redirect write ">"
//             * stdout redirect target "stdout"
//
// $ mlr -n put -v 'tee > "foo.dat", $*'
// DSL EXPRESSION:
// tee > "foo.dat", $*
// RAW AST:
// * statement block
//     * tee statement "tee"
//         * full record "$*"
//         * redirect write ">"
//             * string literal "foo.dat"
//
// $ mlr -n put -v 'tee | "jq .", $*'
// DSL EXPRESSION:
// tee | "jq .", $*
// RAW AST:
// * statement block
//     * tee statement "tee"
//         * full record "$*"
//         * redirect pipe "|"
//             * string literal "jq ."
// ----------------------------------------------------------------

// ================================================================
type tTeeToRedirectFunc func(
	outrec *types.Mlrmap,
	state *State,
) error

type TeeStatementNode struct {
	expressionEvaluable       IEvaluable // always the $* evaluable
	teeToRedirectFunc         tTeeToRedirectFunc
	redirectorTargetEvaluable IEvaluable           // for file/pipe targets
	outputHandlerManager      OutputHandlerManager // for file/pipe targets
}

// ----------------------------------------------------------------
func (this *RootNode) BuildTeeStatementNode(astNode *dsl.ASTNode) (IExecutable, error) {
	lib.InternalCodingErrorIf(astNode.Type != dsl.NodeTypeTeeStatement)
	lib.InternalCodingErrorIf(len(astNode.Children) != 2)
	expressionNode := astNode.Children[0]
	redirectorNode := astNode.Children[1]

	//  - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
	// Expresosin to be teed, which is $*.

	lib.InternalCodingErrorIf(expressionNode.Type != dsl.NodeTypeFullSrec)
	expressionEvaluable := this.BuildFullSrecRvalueNode()

	//  - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
	// Redirection targets (the thing after > >> |, if any).

	retval := &TeeStatementNode{
		expressionEvaluable:       expressionEvaluable,
		teeToRedirectFunc:         nil,
		redirectorTargetEvaluable: nil,
		outputHandlerManager:      nil,
	}

	// There is > >> or | provided.
	lib.InternalCodingErrorIf(redirectorNode.Children == nil)
	lib.InternalCodingErrorIf(len(redirectorNode.Children) != 1)
	redirectorTargetNode := redirectorNode.Children[0]
	var err error = nil

	if redirectorTargetNode.Type == dsl.NodeTypeRedirectTargetStdout {
		retval.teeToRedirectFunc = retval.teeToFileOrPipe
		retval.outputHandlerManager = NewStdoutWriteHandlerManager(this.recordWriterOptions)
		retval.redirectorTargetEvaluable = this.BuildStringLiteralNode("(stdout)")
	} else if redirectorTargetNode.Type == dsl.NodeTypeRedirectTargetStderr {
		retval.teeToRedirectFunc = retval.teeToFileOrPipe
		retval.outputHandlerManager = NewStderrWriteHandlerManager(this.recordWriterOptions)
		retval.redirectorTargetEvaluable = this.BuildStringLiteralNode("(stderr)")
	} else {
		retval.teeToRedirectFunc = retval.teeToFileOrPipe

		retval.redirectorTargetEvaluable, err = this.BuildEvaluableNode(redirectorTargetNode)
		if err != nil {
			return nil, err
		}

		if redirectorNode.Type == dsl.NodeTypeRedirectWrite {
			retval.outputHandlerManager = NewFileWritetHandlerManager(this.recordWriterOptions)
		} else if redirectorNode.Type == dsl.NodeTypeRedirectAppend {
			retval.outputHandlerManager = NewFileAppendHandlerManager(this.recordWriterOptions)
		} else if redirectorNode.Type == dsl.NodeTypeRedirectPipe {
			retval.outputHandlerManager = NewPipeWriteHandlerManager(this.recordWriterOptions)
		} else {
			return nil, errors.New(
				fmt.Sprintf(
					"%s: unhandled redirector node type %s.",
					os.Args[0], string(redirectorNode.Type),
				),
			)
		}
	}

	// Register this with the CST root node so that open file descriptrs can be
	// closed, etc at end of stream.
	if retval.outputHandlerManager != nil {
		this.RegisterOutputHandlerManager(retval.outputHandlerManager)
	}

	return retval, nil
}

// ----------------------------------------------------------------
func (this *TeeStatementNode) Execute(state *State) (*BlockExitPayload, error) {
	evaluation := this.expressionEvaluable.Evaluate(state)
	if !evaluation.IsMap() {
		return nil, errors.New(
			fmt.Sprintf(
				"%s: tee-evaluaiton yielded %s, not map.",
				os.Args[0], evaluation.GetTypeName(),
			),
		)
	}
	this.teeToRedirectFunc(evaluation.GetMap(), state)
	return nil, nil
}

// ----------------------------------------------------------------
func (this *TeeStatementNode) teeToFileOrPipe(
	outrec *types.Mlrmap,
	state *State,
) error {
	redirectorTarget := this.redirectorTargetEvaluable.Evaluate(state)
	if !redirectorTarget.IsString() {
		return errors.New(
			fmt.Sprintf(
				"%s: output redirection yielded %s, not string.",
				os.Args[0], redirectorTarget.GetTypeName(),
			),
		)
	}
	outputFileName := redirectorTarget.String()

	this.outputHandlerManager.WriteRecordAndContext(
		types.NewRecordAndContext(outrec, state.Context),
		outputFileName,
	)
	return nil
}