package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

func getexprs(e ast.Expr) string {
	if lit, ok := e.(*ast.BasicLit); ok {
		return lit.Value
	}
	if ident, ok := e.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func genatomdecl(origfn *ast.FuncDecl, origname, origtag string) (decls []ast.Decl) {
	fieldslist := &ast.FieldList{}
	typespec := &ast.TypeSpec{
		Name: ast.NewIdent(origname),
		Type: &ast.StructType{Fields: fieldslist},
	}

	for _, _stmt := range origfn.Body.List {
		stmt := _stmt.(*ast.ExprStmt)
		callexpr := stmt.X.(*ast.CallExpr)
		typ := callexpr.Fun.(*ast.Ident).Name

		if strings.HasPrefix(typ, "_") {
			if typ == "_unknowns" {
				fieldslist.List = append(fieldslist.List, &ast.Field{
					Names: []*ast.Ident{ast.NewIdent("Unknowns")},
					Type:  ast.NewIdent("[]Atom"),
				})
			}
			continue
		}

		name := getexprs(callexpr.Args[0])

		name2 := ""
		if len(callexpr.Args) > 1 {
			name2 = getexprs(callexpr.Args[1])
		}

		len3 := ""
		if len(callexpr.Args) > 2 {
			len3 = getexprs(callexpr.Args[2])
		}

		if strings.HasPrefix(name, "_") {
			continue
		}

		switch typ {
		case "fixed16":
			typ = "float64"
		case "fixed32":
			typ = "float64"
		case "bytesleft":
			typ = "[]byte"
		case "bytes":
			typ = "[" + name2 + "]byte"
		case "uint24":
			typ = "uint32"
		case "time64", "time32":
			typ = "time.Time"
		case "atom":
			typ = "*" + name2
		case "atoms":
			typ = "[]*" + name2
		case "slice":
			typ = "[]" + name2
		case "array":
			typ = "[" + len3 + "]" + name2
		}

		fieldslist.List = append(fieldslist.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(name)},
			Type:  ast.NewIdent(typ),
		})
	}

	if origtag != "" {
		fieldslist.List = append(fieldslist.List, &ast.Field{
			Type: ast.NewIdent("AtomPos"),
		})
	}

	gendecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			typespec,
		},
	}
	decls = append(decls, gendecl)
	return
}

func typegetlen(typ string) (n int) {
	switch typ {
	case "uint8":
		n = 1
	case "uint16":
		n = 2
	case "uint24":
		n = 3
	case "uint32":
		n = 4
	case "int16":
		n = 2
	case "int32":
		n = 4
	case "uint64":
		n = 8
	case "time32":
		n = 4
	case "time64":
		n = 8
	case "fixed32":
		n = 4
	case "fixed16":
		n = 2
	}
	return
}

func typegetlens(typ string) string {
	n := typegetlen(typ)
	if n == 0 {
		return "Len" + typ
	} else {
		return fmt.Sprint(n)
	}
}

func typegetvartype(typ string) string {
	switch typ {
	case "uint8":
		return "uint8"
	case "uint16":
		return "uint16"
	case "uint24":
		return "uint32"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "int16":
		return "int16"
	case "int32":
		return "int32"
	}
	return ""
}

func typegetputfn(typ string) (fn string) {
	fn = typ
	switch typ {
	case "uint8":
		fn = "pio.PutU8"
	case "uint16":
		fn = "pio.PutU16BE"
	case "uint24":
		fn = "pio.PutU24BE"
	case "uint32":
		fn = "pio.PutU32BE"
	case "int16":
		fn = "pio.PutI16BE"
	case "int32":
		fn = "pio.PutI32BE"
	case "uint64":
		fn = "pio.PutU64BE"
	case "time32":
		fn = "PutTime32"
	case "time64":
		fn = "PutTime64"
	case "fixed32":
		fn = "PutFixed32"
	case "fixed16":
		fn = "PutFixed16"
	default:
		fn = "Put" + typ
	}
	return
}

func typegetgetfn(typ string) (fn string) {
	fn = typ
	switch typ {
	case "uint8":
		fn = "pio.U8"
	case "uint16":
		fn = "pio.U16BE"
	case "uint24":
		fn = "pio.U24BE"
	case "uint32":
		fn = "pio.U32BE"
	case "int16":
		fn = "pio.I16BE"
	case "int32":
		fn = "pio.I32BE"
	case "uint64":
		fn = "pio.U64BE"
	case "time32":
		fn = "GetTime32"
	case "time64":
		fn = "GetTime64"
	case "fixed32":
		fn = "GetFixed32"
	case "fixed16":
		fn = "GetFixed16"
	default:
		fn = "Get" + typ
	}
	return
}

func addns(n string) (stmts []ast.Stmt) {
	assign := &ast.AssignStmt{
		Tok: token.ADD_ASSIGN,
		Lhs: []ast.Expr{ast.NewIdent("n")},
		Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: n}},
	}
	stmts = append(stmts, assign)
	return
}

func addn(n int) (stmts []ast.Stmt) {
	return addns(fmt.Sprint(n))
}

func simplecall(fun string, args ...string) *ast.ExprStmt {
	_args := []ast.Expr{}
	for _, s := range args {
		_args = append(_args, ast.NewIdent(s))
	}
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  ast.NewIdent(fun),
			Args: _args,
		},
	}
}

func getxx(typ string, pos, name string, conv bool) (stmts []ast.Stmt) {
	fn := typegetgetfn(typ)
	assign := &ast.AssignStmt{
		Tok: token.ASSIGN,
		Lhs: []ast.Expr{ast.NewIdent(name)},
		Rhs: []ast.Expr{simplecall(fn, "b["+pos+":]").X},
	}
	stmts = append(stmts, assign)
	return
}

func putxx(typ string, pos, name string, conv bool) (stmts []ast.Stmt) {
	if conv {
		name = fmt.Sprintf("%s(%s)", typ, name)
	}
	fn := typegetputfn(typ)
	stmts = append(stmts, simplecall(fn, "b["+pos+":]", name))
	return
}

func putxxadd(fn string, name string, conv bool) (stmts []ast.Stmt) {
	n := typegetlen(fn)
	stmts = append(stmts, putxx(fn, "n", name, conv)...)
	stmts = append(stmts, addn(n)...)
	return
}

func newdecl(origname, name string, params, res []*ast.Field, stmts []ast.Stmt) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Names: []*ast.Ident{ast.NewIdent("self")},
					Type:  ast.NewIdent(origname),
				},
			},
		},
		Name: ast.NewIdent(name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: params,
			},
			Results: &ast.FieldList{
				List: res,
			},
		},
		Body: &ast.BlockStmt{List: stmts},
	}
}

func getstructputgetlenfn(origfn *ast.FuncDecl, origname string) (decls []ast.Decl) {
	getstmts := []ast.Stmt{}
	putstmts := []ast.Stmt{}
	totlen := 0

	for _, _stmt := range origfn.Body.List {
		stmt := _stmt.(*ast.ExprStmt)
		callexpr := stmt.X.(*ast.CallExpr)
		typ := callexpr.Fun.(*ast.Ident).Name
		name := getexprs(callexpr.Args[0])

		getstmts = append(getstmts, getxx(typ, fmt.Sprint(totlen), "self."+name, false)...)
		putstmts = append(putstmts, putxx(typ, fmt.Sprint(totlen), "self."+name, false)...)
		totlen += typegetlen(typ)
	}

	getstmts = append(getstmts, &ast.ReturnStmt{})

	decls = append(decls, &ast.FuncDecl{
		Name: ast.NewIdent("Get" + origname),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{Names: []*ast.Ident{ast.NewIdent("b")}, Type: ast.NewIdent("[]byte")},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{Names: []*ast.Ident{ast.NewIdent("self")}, Type: ast.NewIdent(origname)},
				},
			},
		},
		Body: &ast.BlockStmt{List: getstmts},
	})

	decls = append(decls, &ast.FuncDecl{
		Name: ast.NewIdent("Put" + origname),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{Names: []*ast.Ident{ast.NewIdent("b")}, Type: ast.NewIdent("[]byte")},
					&ast.Field{Names: []*ast.Ident{ast.NewIdent("self")}, Type: ast.NewIdent(origname)},
				},
			},
		},
		Body: &ast.BlockStmt{List: putstmts},
	})

	decls = append(decls, &ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent("Len" + origname)},
				Values: []ast.Expr{ast.NewIdent(fmt.Sprint(totlen))},
			},
		},
	})

	return
}

func cc4decls(name string) (decls []ast.Decl) {
	constdecl := &ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{
					ast.NewIdent(strings.ToUpper(name)),
				},
				Values: []ast.Expr{
					&ast.CallExpr{
						Fun:  ast.NewIdent("Tag"),
						Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("0x%x", []byte(name))}},
					},
				},
			},
		},
	}
	decls = append(decls, constdecl)
	return
}

func codeclonereplace(stmts []ast.Stmt, doit []ast.Stmt) (out []ast.Stmt) {
	out = append([]ast.Stmt(nil), stmts...)
	for i := range out {
		if ifstmt, ok := out[i].(*ast.IfStmt); ok {
			newifstmt := &ast.IfStmt{
				Cond: ifstmt.Cond,
				Body: &ast.BlockStmt{
					List: codeclonereplace(ifstmt.Body.List, doit),
				},
			}
			if ifstmt.Else != nil {
				newifstmt.Else = &ast.BlockStmt{
					List: codeclonereplace(ifstmt.Else.(*ast.BlockStmt).List, doit),
				}
			}
			out[i] = newifstmt
		} else if exprstmt, ok := out[i].(*ast.ExprStmt); ok {
			if callexpr, ok := exprstmt.X.(*ast.CallExpr); ok {
				if getexprs(callexpr.Fun) == "doit" {
					out[i] = &ast.BlockStmt{List: doit}
				}
			}
		}
	}
	return
}

func getatommarshalfn(origfn *ast.FuncDecl,
	origname, origtag string,
	tagnamemap map[string]string,
) (decls []ast.Decl) {
	marstmts := []ast.Stmt{}
	unmarstmts := []ast.Stmt{}
	lenstmts := []ast.Stmt{}
	childrenstmts := []ast.Stmt{}

	parseerrreturn := func(debug string) (stmts []ast.Stmt) {
		return []ast.Stmt{
			&ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{ast.NewIdent("err")},
				Rhs: []ast.Expr{ast.NewIdent(fmt.Sprintf(`parseErr("%s", n+offset, err)`, debug))},
			},
			&ast.ReturnStmt{},
		}
	}

	callmarshal := func(name string) (stmts []ast.Stmt) {
		callexpr := &ast.CallExpr{
			Fun:  ast.NewIdent(name + ".Marshal"),
			Args: []ast.Expr{ast.NewIdent("b[n:]")},
		}
		assign := &ast.AssignStmt{
			Tok: token.ADD_ASSIGN,
			Lhs: []ast.Expr{ast.NewIdent("n")},
			Rhs: []ast.Expr{callexpr},
		}
		stmts = append(stmts, assign)
		return
	}

	callputstruct := func(typ, name string) (stmts []ast.Stmt) {
		stmts = append(stmts, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  ast.NewIdent(typegetputfn(typ)),
				Args: []ast.Expr{ast.NewIdent("b[n:]"), ast.NewIdent(name)},
			},
		})
		stmts = append(stmts, &ast.AssignStmt{
			Tok: token.ADD_ASSIGN,
			Lhs: []ast.Expr{ast.NewIdent("n")},
			Rhs: []ast.Expr{ast.NewIdent(typegetlens(typ))},
		})
		return
	}

	calllenstruct := func(typ, name string) (stmts []ast.Stmt) {
		inc := typegetlens(typ) + "*len(" + name + ")"
		stmts = append(stmts, &ast.AssignStmt{
			Tok: token.ADD_ASSIGN,
			Lhs: []ast.Expr{ast.NewIdent("n")},
			Rhs: []ast.Expr{ast.NewIdent(inc)},
		})
		return
	}

	calllen := func(name string) (stmts []ast.Stmt) {
		callexpr := &ast.CallExpr{
			Fun:  ast.NewIdent(name + ".Len"),
			Args: []ast.Expr{},
		}
		assign := &ast.AssignStmt{
			Tok: token.ADD_ASSIGN,
			Lhs: []ast.Expr{ast.NewIdent("n")},
			Rhs: []ast.Expr{callexpr},
		}
		stmts = append(stmts, assign)
		return
	}

	foreach := func(name, field string, block []ast.Stmt) (stmts []ast.Stmt) {
		rangestmt := &ast.RangeStmt{
			Key:   ast.NewIdent("_"),
			Value: ast.NewIdent(name),
			Body: &ast.BlockStmt{
				List: block,
			},
			Tok: token.DEFINE,
			X:   ast.NewIdent(field),
		}
		stmts = append(stmts, rangestmt)
		return
	}

	foreachatom := func(field string, block []ast.Stmt) (stmts []ast.Stmt) {
		return foreach("atom", field, block)
	}

	foreachentry := func(field string, block []ast.Stmt) (stmts []ast.Stmt) {
		return foreach("entry", field, block)
	}

	foreachi := func(field string, block []ast.Stmt) (stmts []ast.Stmt) {
		rangestmt := &ast.RangeStmt{
			Key: ast.NewIdent("i"),
			Body: &ast.BlockStmt{
				List: block,
			},
			Tok: token.DEFINE,
			X:   ast.NewIdent(field),
		}
		stmts = append(stmts, rangestmt)
		return
	}

	foreachunknowns := func(block []ast.Stmt) (stmts []ast.Stmt) {
		return foreachatom("self.Unknowns", block)
	}

	declvar := func(name, typ string) (stmts []ast.Stmt) {
		stmts = append(stmts, &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{
							ast.NewIdent(typ),
						},
						Type: ast.NewIdent(name),
					},
				},
			},
		})
		return
	}

	makeslice := func(name, typ, size string) (stmts []ast.Stmt) {
		stmts = append(stmts, &ast.ExprStmt{
			X: ast.NewIdent(fmt.Sprintf("%s = make([]%s, %s)", name, typ, size)),
		})
		return
	}

	simpleassign := func(tok token.Token, l, r string) *ast.AssignStmt {
		return &ast.AssignStmt{
			Tok: tok,
			Lhs: []ast.Expr{ast.NewIdent(l)},
			Rhs: []ast.Expr{ast.NewIdent(r)},
		}
	}

	struct2tag := func(s string) string {
		name := tagnamemap[s]
		return name
	}

	foreachatomsappendchildren := func(field string) (stmts []ast.Stmt) {
		return foreachatom(field, []ast.Stmt{
			simpleassign(token.ASSIGN, "r", "append(r, atom)"),
		})
	}

	var hasunknowns bool
	var atomnames []string
	var atomtypes []string
	var atomarrnames []string
	var atomarrtypes []string
	slicenamemap := map[string]string{}

	unmarshalatom := func(typ, init string) (stmts []ast.Stmt) {
		return []ast.Stmt{
			&ast.AssignStmt{Tok: token.DEFINE,
				Lhs: []ast.Expr{ast.NewIdent("atom")}, Rhs: []ast.Expr{ast.NewIdent("&" + typ + "{" + init + "}")},
			},
			&ast.IfStmt{
				Init: &ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{ast.NewIdent("_"), ast.NewIdent("err")},
					Rhs: []ast.Expr{ast.NewIdent("atom.Unmarshal(b[n:n+size], offset+n)")},
				},
				Cond: ast.NewIdent("err != nil"),
				Body: &ast.BlockStmt{List: parseerrreturn(struct2tag(typ))},
			},
		}
	}

	unmrashalatoms := func() (stmts []ast.Stmt) {
		blocks := []ast.Stmt{}

		blocks = append(blocks, &ast.AssignStmt{Tok: token.DEFINE, Lhs: []ast.Expr{ast.NewIdent("tag")},
			Rhs: []ast.Expr{ast.NewIdent("Tag(pio.U32BE(b[n+4:]))")},
		})
		blocks = append(blocks, &ast.AssignStmt{Tok: token.DEFINE, Lhs: []ast.Expr{ast.NewIdent("size")},
			Rhs: []ast.Expr{ast.NewIdent("int(pio.U32BE(b[n:]))")},
		})
		blocks = append(blocks, &ast.IfStmt{
			Cond: ast.NewIdent("len(b) < n+size"),
			Body: &ast.BlockStmt{List: parseerrreturn("TagSizeInvalid")},
		})

		cases := []ast.Stmt{}

		for i, atom := range atomnames {
			cases = append(cases, &ast.CaseClause{
				List: []ast.Expr{ast.NewIdent(strings.ToUpper(struct2tag(atomtypes[i])))},
				Body: []ast.Stmt{&ast.BlockStmt{
					List: append(unmarshalatom(atomtypes[i], ""), simpleassign(token.ASSIGN, "self."+atom, "atom")),
				}},
			})
		}

		for i, atom := range atomarrnames {
			selfatom := "self." + atom
			cases = append(cases, &ast.CaseClause{
				List: []ast.Expr{ast.NewIdent(strings.ToUpper(struct2tag(atomarrtypes[i])))},
				Body: []ast.Stmt{&ast.BlockStmt{
					List: append(unmarshalatom(atomarrtypes[i], ""),
						simpleassign(token.ASSIGN, selfatom, "append("+selfatom+", atom)")),
				}},
			})
		}

		if hasunknowns {
			init := "Tag_: tag, Data: b[n:n+size]"
			selfatom := "self.Unknowns"
			cases = append(cases, &ast.CaseClause{
				Body: []ast.Stmt{&ast.BlockStmt{
					List: append(unmarshalatom("Dummy", init), simpleassign(token.ASSIGN, selfatom, "append("+selfatom+", atom)")),
				}},
			})
		}

		blocks = append(blocks, &ast.SwitchStmt{
			Tag:  ast.NewIdent("tag"),
			Body: &ast.BlockStmt{List: cases},
		})

		blocks = append(blocks, addns("size")...)

		stmts = append(stmts, &ast.ForStmt{
			Cond: ast.NewIdent("n+8 < len(b)"),
			Body: &ast.BlockStmt{List: blocks},
		})
		return
	}

	marshalwrapstmts := func() (stmts []ast.Stmt) {
		stmts = append(stmts, putxx("uint32", "4", strings.ToUpper(origtag), true)...)
		stmts = append(stmts, addns("self.marshal(b[8:])+8")...)
		stmts = append(stmts, putxx("uint32", "0", "n", true)...)
		stmts = append(stmts, &ast.ReturnStmt{})
		return
	}

	ifnotnil := func(name string, block []ast.Stmt) (stmts []ast.Stmt) {
		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  ast.NewIdent(name),
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			Body: &ast.BlockStmt{List: block},
		})
		return
	}

	getchildrennr := func(name string) (stmts []ast.Stmt) {
		stmts = append(stmts, &ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{ast.NewIdent(name)},
			Rhs: []ast.Expr{ast.NewIdent("0")},
		})
		for _, atom := range atomnames {
			stmts = append(stmts, ifnotnil("self."+atom, []ast.Stmt{
				&ast.IncDecStmt{X: ast.NewIdent(name), Tok: token.INC},
			})...)
		}
		if hasunknowns {
			assign := &ast.AssignStmt{
				Tok: token.ADD_ASSIGN,
				Lhs: []ast.Expr{ast.NewIdent("_childrenNR")},
				Rhs: []ast.Expr{ast.NewIdent("len(self.Unknowns)")},
			}
			stmts = append(stmts, assign)
		}
		return
	}

	checkcurlen := func(inc, debug string) (stmts []ast.Stmt) {
		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  ast.NewIdent("len(b)"),
				Op: token.LSS,
				Y:  ast.NewIdent("n+" + inc),
			},
			Body: &ast.BlockStmt{List: parseerrreturn(debug)},
		})
		return
	}

	checklendo := func(typ, name, debug string) (stmts []ast.Stmt) {
		stmts = append(stmts, checkcurlen(typegetlens(typ), debug)...)
		stmts = append(stmts, getxx(typ, "n", name, false)...)
		stmts = append(stmts, addns(typegetlens(typ))...)
		return
	}

	checkstructlendo := func(typ, name, debug string,
		foreach func(string, []ast.Stmt) []ast.Stmt,
	) (stmts []ast.Stmt) {
		inc := typegetlens(typ) + "*len(" + name + ")"
		stmts = append(stmts, checkcurlen(inc, debug)...)
		stmts = append(stmts, foreach(name, append(
			[]ast.Stmt{
				&ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{
						ast.NewIdent(name + "[i]"),
					},
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun:  ast.NewIdent(typegetgetfn(typ)),
							Args: []ast.Expr{ast.NewIdent("b[n:]")},
						},
					},
				},
			},
			addns(typegetlens(typ))...,
		))...)
		return
	}

	checklencopy := func(name string) (stmts []ast.Stmt) {
		lens := fmt.Sprintf("len(self.%s)", name)
		stmts = append(stmts, checkcurlen(lens, name)...)
		stmts = append(stmts, simplecall("copy", fmt.Sprintf("self.%s[:]", name), "b[n:]"))
		stmts = append(stmts, addns(lens)...)
		return
	}

	appendcode := func(args []ast.Expr,
		marstmts *[]ast.Stmt, lenstmts *[]ast.Stmt, unmarstmts *[]ast.Stmt,
		defmarstmts []ast.Stmt, deflenstmts []ast.Stmt, defunmarstmts []ast.Stmt,
	) {
		bodylist := func(i int, doit []ast.Stmt) []ast.Stmt {
			return codeclonereplace(args[i].(*ast.FuncLit).Body.List, doit)
		}
		if len(args) == 1 {
			*marstmts = append(*marstmts, bodylist(0, defmarstmts)...)
			*lenstmts = append(*lenstmts, bodylist(0, deflenstmts)...)
			*unmarstmts = append(*unmarstmts, bodylist(0, defunmarstmts)...)
		} else {
			*marstmts = append(*marstmts, bodylist(0, defmarstmts)...)
			*lenstmts = append(*lenstmts, bodylist(1, deflenstmts)...)
			*unmarstmts = append(*unmarstmts, bodylist(2, defunmarstmts)...)
		}
	}

	getdefaultstmts := func(
		typ, name, name2 string,
		marstmts *[]ast.Stmt, lenstmts *[]ast.Stmt,
		unmarstmts *[]ast.Stmt, childrenstmts *[]ast.Stmt,
	) {
		switch typ {
		case "bytes", "bytesleft":
			*marstmts = append(*marstmts, simplecall("copy", "b[n:]", "self."+name+"[:]"))
			*marstmts = append(*marstmts, addns(fmt.Sprintf("len(self.%s[:])", name))...)
			*lenstmts = append(*lenstmts, addns(fmt.Sprintf("len(self.%s[:])", name))...)
			if typ == "bytes" {
				*unmarstmts = append(*unmarstmts, checklencopy(name)...)
			} else {
				*unmarstmts = append(*unmarstmts, simpleassign(token.ASSIGN, "self."+name, "b[n:]"))
				*unmarstmts = append(*unmarstmts, addns("len(b[n:])")...)
			}

		case "array":
			*marstmts = append(*marstmts, foreachentry("self."+name, callputstruct(name2, "entry"))...)
			*lenstmts = append(*lenstmts, calllenstruct(name2, "self."+name+"[:]")...)
			*unmarstmts = append(*unmarstmts, checkstructlendo(name2, "self."+name, name, foreachi)...)

		case "atoms":
			*marstmts = append(*marstmts, foreachatom("self."+name, callmarshal("atom"))...)
			*lenstmts = append(*lenstmts, foreachatom("self."+name, calllen("atom"))...)
			*childrenstmts = append(*childrenstmts, foreachatomsappendchildren("self."+name)...)

		case "slice":
			*marstmts = append(*marstmts, foreachentry("self."+name, callputstruct(name2, "entry"))...)
			*lenstmts = append(*lenstmts, calllenstruct(name2, "self."+name)...)
			*unmarstmts = append(*unmarstmts, checkstructlendo(name2, "self."+name, name2, foreachi)...)

		case "atom":
			*marstmts = append(*marstmts, ifnotnil("self."+name, callmarshal("self."+name))...)
			*lenstmts = append(*lenstmts, ifnotnil("self."+name, calllen("self."+name))...)
			*childrenstmts = append(*childrenstmts, ifnotnil("self."+name, []ast.Stmt{
				simpleassign(token.ASSIGN, "r", fmt.Sprintf("append(r, %s)", "self."+name)),
			})...)

		default:
			*marstmts = append(*marstmts, putxxadd(typ, "self."+name, false)...)
			*lenstmts = append(*lenstmts, addn(typegetlen(typ))...)
			*unmarstmts = append(*unmarstmts, checklendo(typ, "self."+name, name)...)
		}
	}

	for _, _stmt := range origfn.Body.List {
		stmt := _stmt.(*ast.ExprStmt)
		callexpr := stmt.X.(*ast.CallExpr)
		typ := callexpr.Fun.(*ast.Ident).Name
		if typ == "_unknowns" {
			hasunknowns = true
		} else if typ == "atom" {
			name := getexprs(callexpr.Args[0])
			name2 := getexprs(callexpr.Args[1])
			atomnames = append(atomnames, name)
			atomtypes = append(atomtypes, name2)
		} else if typ == "atoms" {
			name := getexprs(callexpr.Args[0])
			name2 := getexprs(callexpr.Args[1])
			atomarrnames = append(atomarrnames, name)
			atomarrtypes = append(atomarrtypes, name2)
		} else if typ == "slice" {
			name := getexprs(callexpr.Args[0])
			name2 := getexprs(callexpr.Args[1])
			slicenamemap[name] = name2
		}
	}

	lenstmts = append(lenstmts, addn(8)...)
	unmarstmts = append(unmarstmts, simplecall("(&self.AtomPos).setPos", "offset", "len(b)"))
	unmarstmts = append(unmarstmts, addn(8)...)

	for _, _stmt := range origfn.Body.List {
		stmt := _stmt.(*ast.ExprStmt)
		callexpr := stmt.X.(*ast.CallExpr)
		typ := callexpr.Fun.(*ast.Ident).Name

		name := ""
		if len(callexpr.Args) > 0 {
			name = getexprs(callexpr.Args[0])
		}

		name2 := ""
		if len(callexpr.Args) > 1 {
			name2 = getexprs(callexpr.Args[1])
		}

		var defmarstmts, deflenstmts, defunmarstmts, defchildrenstmts []ast.Stmt
		getdefaultstmts(typ, name, name2,
			&defmarstmts, &deflenstmts, &defunmarstmts, &defchildrenstmts)

		var code []ast.Expr
		for _, arg := range callexpr.Args {
			if fn, ok := arg.(*ast.CallExpr); ok {
				if getexprs(fn.Fun) == "_code" {
					code = fn.Args
				}
			}
		}
		if code != nil {
			appendcode(code,
				&marstmts, &lenstmts, &unmarstmts,
				defmarstmts, deflenstmts, defunmarstmts,
			)
			continue
		}

		if strings.HasPrefix(typ, "_") {
			if typ == "_unknowns" {
				marstmts = append(marstmts, foreachunknowns(callmarshal("atom"))...)
				lenstmts = append(lenstmts, foreachunknowns(calllen("atom"))...)
				childrenstmts = append(childrenstmts, simpleassign(token.ASSIGN, "r", "append(r, self.Unknowns...)"))
			}
			if typ == "_skip" {
				marstmts = append(marstmts, addns(name)...)
				lenstmts = append(lenstmts, addns(name)...)
				unmarstmts = append(unmarstmts, addns(name)...)
			}
			if typ == "_code" {
				appendcode(callexpr.Args,
					&marstmts, &lenstmts, &unmarstmts,
					defmarstmts, deflenstmts, defunmarstmts,
				)
			}
			continue
		}

		if name == "_childrenNR" {
			marstmts = append(marstmts, getchildrennr(name)...)
			marstmts = append(marstmts, putxxadd(typ, name, true)...)
			lenstmts = append(lenstmts, addn(typegetlen(typ))...)
			unmarstmts = append(unmarstmts, addn(typegetlen(typ))...)
			continue
		}

		if strings.HasPrefix(name, "_len_") {
			field := name[len("_len_"):]
			marstmts = append(marstmts, putxxadd(typ, "len(self."+field+")", true)...)
			lenstmts = append(lenstmts, addn(typegetlen(typ))...)
			unmarstmts = append(unmarstmts, declvar(typegetvartype(typ), name)...)
			unmarstmts = append(unmarstmts, getxx(typ, "n", name, false)...)
			unmarstmts = append(unmarstmts, addn(typegetlen(typ))...)
			unmarstmts = append(unmarstmts, makeslice("self."+field, slicenamemap[field], name)...)
			continue
		}

		marstmts = append(marstmts, defmarstmts...)
		lenstmts = append(lenstmts, deflenstmts...)
		unmarstmts = append(unmarstmts, defunmarstmts...)
		childrenstmts = append(childrenstmts, defchildrenstmts...)
	}

	if len(atomnames) > 0 || len(atomarrnames) > 0 || hasunknowns {
		unmarstmts = append(unmarstmts, unmrashalatoms()...)
	}

	marstmts = append(marstmts, &ast.ReturnStmt{})
	lenstmts = append(lenstmts, &ast.ReturnStmt{})
	unmarstmts = append(unmarstmts, &ast.ReturnStmt{})
	childrenstmts = append(childrenstmts, &ast.ReturnStmt{})

	decls = append(decls, newdecl(origname, "Marshal", []*ast.Field{
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("b")}, Type: ast.NewIdent("[]byte")},
	}, []*ast.Field{
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("n")}, Type: ast.NewIdent("int")},
	}, marshalwrapstmts()))

	decls = append(decls, newdecl(origname, "marshal", []*ast.Field{
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("b")}, Type: ast.NewIdent("[]byte")},
	}, []*ast.Field{
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("n")}, Type: ast.NewIdent("int")},
	}, marstmts))

	decls = append(decls, newdecl(origname, "Len", []*ast.Field{}, []*ast.Field{
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("n")}, Type: ast.NewIdent("int")},
	}, lenstmts))

	decls = append(decls, newdecl("*"+origname, "Unmarshal", []*ast.Field{
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("b")}, Type: ast.NewIdent("[]byte")},
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("offset")}, Type: ast.NewIdent("int")},
	}, []*ast.Field{
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("n")}, Type: ast.NewIdent("int")},
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("err")}, Type: ast.NewIdent("error")},
	}, unmarstmts))

	decls = append(decls, newdecl(origname, "Children", []*ast.Field{}, []*ast.Field{
		&ast.Field{Names: []*ast.Ident{ast.NewIdent("r")}, Type: ast.NewIdent("[]Atom")},
	}, childrenstmts))

	return
}

func genatoms(filename, outfilename string) {
	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		panic(err)
	}

	gen := &ast.File{}
	gen.Name = ast.NewIdent("mp4io")
	gen.Decls = []ast.Decl{
		&ast.GenDecl{
			Tok: token.IMPORT,
			Specs: []ast.Spec{
				&ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: `"github.com/nareix/joy4/utils/bits/pio"`}},
			},
		},
		&ast.GenDecl{
			Tok: token.IMPORT,
			Specs: []ast.Spec{
				&ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: `"time"`}},
			},
		},
	}

	tagnamemap := map[string]string{}
	tagnamemap["ElemStreamDesc"] = "esds"

	splittagname := func(fnname string) (ok bool, tag, name string) {
		if len(fnname) > 5 && fnname[4] == '_' {
			tag = fnname[0:4]
			tag = strings.Replace(tag, "_", " ", 1)
			name = fnname[5:]
			ok = true
		} else {
			name = fnname
		}
		return
	}

	for _, decl := range file.Decls {
		if fndecl, ok := decl.(*ast.FuncDecl); ok {
			ok, tag, name := splittagname(fndecl.Name.Name)
			if ok {
				tagnamemap[name] = tag
			}
		}
	}

	tagfuncdecl := func(name, tag string) (decls ast.Decl) {
		return newdecl(name, "Tag", []*ast.Field{}, []*ast.Field{
			&ast.Field{Type: ast.NewIdent("Tag")},
		}, []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{ast.NewIdent(strings.ToUpper(tag))}}})
	}

	for k, v := range tagnamemap {
		gen.Decls = append(gen.Decls, cc4decls(v)...)
		gen.Decls = append(gen.Decls, tagfuncdecl(k, v))
	}
	gen.Decls = append(gen.Decls, cc4decls("mdat")...)

	for _, decl := range file.Decls {
		if fndecl, ok := decl.(*ast.FuncDecl); ok {
			ok, tag, name := splittagname(fndecl.Name.Name)
			if ok {
				gen.Decls = append(gen.Decls, genatomdecl(fndecl, name, tag)...)
				gen.Decls = append(gen.Decls, getatommarshalfn(fndecl, name, tag, tagnamemap)...)
			} else {
				gen.Decls = append(gen.Decls, genatomdecl(fndecl, name, tag)...)
				gen.Decls = append(gen.Decls, getstructputgetlenfn(fndecl, name)...)
			}
		}
	}

	outfile, _ := os.Create(outfilename)
	printer.Fprint(outfile, fset, gen)
	outfile.Close()
}

func parse(filename, outfilename string) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		panic(err)
	}
	outfile, _ := os.Create(outfilename)
	ast.Fprint(outfile, fset, file, nil)
	outfile.Close()
}

func main() {
	switch os.Args[1] {
	case "parse":
		parse(os.Args[2], os.Args[3])

	case "gen":
		genatoms(os.Args[2], os.Args[3])
	}
}
