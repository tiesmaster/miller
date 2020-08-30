package parsing

import (
	"fmt"
	"miller/parsing/token"
)

type TNodeType string

const (
	NodeTypeStringLiteral  = "StringLiteral"
	NodeTypeNumberLiteral  = "NumberLiteral"
	NodeTypeBooleanLiteral = "BooleanLiteral"

	NodeTypeDirectFieldName   = "DirectFieldName"
	NodeTypeIndirectFieldName = "IndirectFieldName"

	NodeTypeStatementBlock  = "StatementBlock"
	NodeTypeStatement       = "Statement"
	NodeTypeAssignment      = "Assignment"
	NodeTypeOperator        = "Operator"
	NodeTypeFieldName       = "FieldName"
	NodeTypeContextVariable = "ContextVariable"
)

// ----------------------------------------------------------------
// xxx comment interface{} everywhere vs. true types due to gocc polymorphism API.
// and, line-count for casts here vs in the BNF:
//
// Statement :
//   md_token_field_name md_token_assign md_token_number
//
// Statement :
//   md_token_field_name md_token_assign md_token_number
//     << dsl.NewASTNodeTernary("foo", $0, $1, $2) >> ;

// ----------------------------------------------------------------
type AST struct {
	Root *ASTNode
}

func NewAST(root interface{}) (*AST, error) {
	return &AST{
		root.(*ASTNode),
	}, nil
}

func (this *AST) Print() {
	this.Root.Print(0)
}

// ----------------------------------------------------------------
type ASTNode struct {
	Token    *token.Token // Nil for tokenless/structural nodes
	NodeType TNodeType
	Children []*ASTNode

	// xxx sketch:
	// * no longer have separate AST/CST as in the C version ?
	// * have a nullable evaluator function pointer attached to each node
	// * outrec := node.Evaluate(inrec, state) ?
	// * what about evaluator-state ?
	// * outrec := node.Evaluator.Evaluate(inrec, state) ?
	// * state:
	//   o lhmsmv_t*        ptyped_overlay; -- get rid of, w/ mlrval keys directly in the lrecs?
	//   o string_array_t** ppregex_captures;
	//   o mlhmmv_root_t*   poosvars;
	//   o context_t*       pctx;
	//   o local_stack_t*   plocal_stack;
	//   o loop_stack_t*    ploop_stack;
	//   o return_state_t   return_state;
	//   o int              trace_execution; -- move out of state?
	//   o int              json_quote_int_keys; -- move out of state?
	//   o int              json_quote_non_string_values; -- move out of state?
	// * statements:
	// * node.Executor.Execute(inrec, state) ?
}

func (this *ASTNode) Print(depth int) {
	for i := 0; i < depth; i++ {
		fmt.Print("    ")
	}
	tok := this.Token
	fmt.Print("* " + this.NodeType)

	if tok != nil {
		fmt.Printf(" \"%s\" \"%s\"",
			token.TokMap.Id(tok.Type), string(tok.Lit))
	}
	fmt.Println()
	if this.Children != nil {
		for _, child := range this.Children {
			child.Print(depth + 1)
		}
	}
}

func NewASTNode(itok interface{}, nodeType TNodeType) (*ASTNode, error) {
	return NewASTNodeNestable(itok, nodeType), nil
}

// xxx comment why grammar use
func NewASTNodeNestable(itok interface{}, nodeType TNodeType) *ASTNode {
	var tok *token.Token = nil
	if itok != nil {
		tok = itok.(*token.Token)
	}
	return &ASTNode{
		tok,
		nodeType,
		nil,
	}
}

func NewASTNodeUnary(itok, childA interface{}, nodeType TNodeType) (*ASTNode, error) {
	parent := NewASTNodeNestable(itok, nodeType)
	convertToUnary(parent, childA)
	return parent, nil
}

// Signature: Token Node Node Type
func NewASTNodeBinary(itok, childA, childB interface{}, nodeType TNodeType) (*ASTNode, error) {
	parent := NewASTNodeNestable(itok, nodeType)
	convertToBinary(parent, childA, childB)
	return parent, nil
}

func NewASTNodeTernary(itok, childA, childB, childC interface{}, nodeType TNodeType) (*ASTNode, error) {
	parent := NewASTNodeNestable(itok, nodeType)
	convertToTernary(parent, childA, childB, childC)
	return parent, nil
}

// Pass-through expressions in the grammar sometimes need to be turned from
// (ASTNode) to (ASTNode, error)
func Nestable(iparent interface{}) (*ASTNode, error) {
	return iparent.(*ASTNode), nil
}

func convertToZary(iparent interface{}) {
	parent := iparent.(*ASTNode)
	children := make([]*ASTNode, 0)
	parent.Children = children
}

func convertToUnary(iparent interface{}, childA interface{}) {
	parent := iparent.(*ASTNode)
	children := make([]*ASTNode, 1)
	children[0] = childA.(*ASTNode)
	parent.Children = children
}

func convertToBinary(iparent interface{}, childA, childB interface{}) {
	parent := iparent.(*ASTNode)
	children := make([]*ASTNode, 2)
	children[0] = childA.(*ASTNode)
	children[1] = childB.(*ASTNode)
	parent.Children = children
}

func convertToTernary(iparent interface{}, childA, childB, childC interface{}) {
	parent := iparent.(*ASTNode)
	children := make([]*ASTNode, 3)
	children[0] = childA.(*ASTNode)
	children[1] = childB.(*ASTNode)
	children[2] = childC.(*ASTNode)
	parent.Children = children
}

func AppendChild(iparent interface{}, child interface{}) (*ASTNode, error) {
	parent := iparent.(*ASTNode)
	if parent.Children == nil {
		convertToUnary(iparent, child)
	} else {
		parent.Children = append(parent.Children, child.(*ASTNode))
	}
	return parent, nil
}