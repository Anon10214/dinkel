from antlr4 import *
from antlr4.tree.Tree import TerminalNode
from CypherLexer import CypherLexer
from CypherParser import CypherParser
from CypherListener import CypherListener

class AnalysisUnit():
    def __init__(self,  query):
        self.query = query 
    
    @staticmethod
    def header_row() -> str:
        keyword_str = ','.join([i.lower()+'_count' for i in terminal_names])
        return f'target,query,size,data_dependencies,qc,ags,max_depth,keyword_count,{keyword_str},target'

    def to_row(self) -> str:
        global max_depth, cur_depth, terminals, qc
        '''
        query,size,keyword count,data dependencies,qc,ags,max depth,count of single keywords...
        '''
        query = self.query.replace('"', '""')

        size = len(self.query)

        dependencies.clear()
        qc.clear()

        analyze(self.query)

        deps = sum([val for _, val in dependencies.items()])
        qcs = sum([val for i, val in dependencies.items() if i in qc])

        terminals_copy = terminals
        terminals = (CypherLexer.FALSE - CypherLexer.UNION + 1) * [0]

        tmp = max_depth
        max_depth = 1
        cur_depth = 1

        terminals_str = ','.join([str(i) for i in terminals_copy])
        return f'"{query}",{size},{deps},{qcs},{deps-qcs},{tmp},{sum(terminals_copy)},{terminals_str}'

terminals = (CypherLexer.FALSE - CypherLexer.UNION + 1) * [0]
terminal_names = CypherLexer.ruleNames[CypherLexer.UNION - 1:CypherLexer.FALSE]

max_depth = 1
cur_depth = 1

qc = []
dependencies = {}


def analyze(query: str):
    input_stream = InputStream(query)
    lexer = CypherLexer(input_stream)
    stream = CommonTokenStream(lexer)
    parser = CypherParser(stream)
    tree = parser.oC_Cypher()
    printer = CypherWalker()
    walker = ParseTreeWalker()
    walker.walk(printer, tree)

def enter_depth():
    global max_depth, cur_depth
    cur_depth += 1
    if cur_depth > max_depth:
        max_depth = cur_depth

def exit_depth():
    global cur_depth
    cur_depth -= 1

class CypherWalker(CypherListener):
    def exitOC_Variable(self, ctx):
        if ctx.getText() not in dependencies:
            qc.append(ctx.getText())
            dependencies[ctx.getText()] = -1
        dependencies[ctx.getText()] += 1

    def exitOC_NodeLabel(self, ctx):
        if ctx.getText() not in dependencies:
            dependencies[ctx.getText()] = -1
        dependencies[ctx.getText()] += 1

    def exitOC_RelTypeName(self, ctx):
        if ctx.getText() not in dependencies:
            dependencies[ctx.getText()] = -1
        dependencies[ctx.getText()] += 1

    def exitOC_PropertyKeyName(self, ctx):
        if ctx.getText() not in dependencies:
            dependencies[ctx.getText()] = -1
        dependencies[ctx.getText()] += 1
    
    # ----------- Clauses for Context Depth -----------

    def enterOC_CallSubquery(self, ctx):
        enter_depth()
    def exitOC_CallSubquery(self, ctx):
        exit_depth()

    def enterOC_CountSubquery(self, ctx):
        enter_depth()
    def exitOC_CountSubquery(self, ctx):
        exit_depth()

    def enterOC_ExistentialSubquery(self, ctx):
        enter_depth()
    def exitOC_ExistentialSubquery(self, ctx):
        exit_depth()

    def enterOC_Foreach(self, ctx):
        enter_depth()
    def exitOC_Foreach(self, ctx):
        exit_depth()

    def enterOC_ListPredicateExpression(self, ctx):
        enter_depth()
    def exitOC_ListPredicateExpression(self, ctx):
        exit_depth()

    # Predicates
    def enterOC_Quantifier(self, ctx):
        enter_depth()
    def exitOC_Quantifier(self, ctx):
        exit_depth()

    def visitTerminal(self, node: TerminalNode):
        if CypherLexer.UNION <= node.symbol.type <= CypherLexer.FALSE:
            terminals[node.symbol.type - CypherLexer.UNION] += 1