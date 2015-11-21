
var uc = x => x && x.substr(0,1).toUpperCase()+x.slice(1);
Array.prototype.nonull = function () {
	return this.filter(x => x);
};

var atoms = {
	fileType: {
		cc4: 'ftyp',
		fields: [
		],
	},

	movie: {
		cc4: 'moov',
		atoms: [
			['header', '*movieHeader'],
			['tracks', '[]*track'],
		],
	},

	movieHeader: {
		cc4: 'mvhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['cTime', 'TimeStamp32'],
			['mTime', 'TimeStamp32'],
			['timeScale', 'TimeStamp32'],
			['duration', 'TimeStamp32'],
			['preferredRate', 'int32'],
			['preferredVolume', 'int16'],
			['_', '[10]byte'],
			['matrix', '[9]int32'],
			['previewTime', 'TimeStamp32'],
			['previewDuration', 'TimeStamp32'],
			['posterTime', 'TimeStamp32'],
			['selectionTime', 'TimeStamp32'],
			['selectionDuration', 'TimeStamp32'],
			['currentTime', 'TimeStamp32'],
			['nextTrackId', 'int32'],
		],
	},

	track: {
		cc4: 'trak',
		atoms: [
			['header', '*trackHeader'],
			['media', '*media'],
		],
	},

	trackHeader: {
		cc4: 'tkhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['cTime', 'TimeStamp32'],
			['mTime', 'TimeStamp32'],
			['trackId', 'TimeStamp32'],
			['_', '[4]byte'],
			['duration', 'TimeStamp32'],
			['_', '[8]byte'],
			['layer', 'int16'],
			['alternateGroup', 'int16'],
			['volume', 'int16'],
			['_', '[2]byte'],
			['matrix', '[9]int32'],
			['trackWidth', 'int32'],
			['trackHeader', 'int32'],
		],
	},

	media: {
		cc4: 'mdia',
		atoms: [
			['header', '*mediaHeader'],
			['info', '*mediaInfo'],
		],
	},

	mediaHeader: {
		cc4: 'mdhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['cTime', 'int32'],
			['mTime', 'int32'],
			['timeScale', 'int32'],
			['duration', 'int32'],
			['language', 'int16'],
			['quality', 'int16'],
		],
	},

	mediaInfo: {
		cc4: 'minf',
		atoms: [
			['sound', '*soundMediaInfo'],
			['video', '*videoMediaInfo'],
			['sample', '*sampleTable'],
		],
	},

	soundMediaInfo: {
		cc4: 'smhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['balance', 'int16'],
			['_', 'int16'],
		],
	},

	videoMediaInfo: {
		cc4: 'vmhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['graphicsMode', 'int16'],
			['opcolor', '[3]int16'],
		],
	},

	sampleTable: {
		cc4: 'stbl',
		atoms: [
			['sampleDesc', '*sampleDesc'],
			['timeToSample', '*timeToSample'],
			['compositionOffset', '*compositionOffset'],
			['sampleToChunk', '*sampleToChunk'],
			['syncSample', '*syncSample'],
			['chunkOffset', '*chunkOffset'],
			['sampleSize', '*sampleSize'],
		],
	},

	sampleDesc: {
		cc4: 'stsd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['entries', '[int32]*sampleDescEntry'],
		],
	},

	timeToSample: {
		cc4: 'stts',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['entries', '[int32]timeToSampleEntry'],
		],
	},

	timeToSampleEntry: {
		fields: [
			['count', 'int32'],
			['duration', 'int32'],
		],
	},

	sampleToChunk: {
		cc4: 'stsc',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['entries', '[int32]sampleToChunkEntry'],
		],
	},

	sampleToChunkEntry: {
		fields: [
			['firstChunk', 'int32'],
			['samplesPerChunk', 'int32'],
			['sampleDescId', 'int32'],
		],
	},

	compositionOffset: {
		cc4: 'ctts',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['entries', '[int32]int32'],
		],
	},

	compositionOffsetEntry: {
		fields: [
			['count', 'int32'],
			['offset', 'int32'],
		],
	},

	syncSample: {
		cc4: 'stss',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['entries', '[int32]int32'],
		],
	},

	sampleSize: {
		cc4: 'stsz',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['entries', '[int32]int32'],
		],
	},

	chunkOffset: {
		cc4: 'stco',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['entries', '[int32]int32'],
		],
	},

};

var DeclReadFunc = (opts) => {
	var stmts = [];

	var DebugStmt = type => StrStmt(`// ${JSON.stringify(type)}`);

	var ReadArr = (name, type) => {
		return [
			//StrStmt('// ReadArr'),
			//DebugStmt(type),
			type.varcount && [
				DeclVar('count', 'int'),
				CallCheckAssign('ReadInt', ['r', type.varcount], ['count']),
				StrStmt(`${name} = make(${typeStr(type)}, count)`),
			],
			For(RangeN('i', type.varcount ? 'count' : type.count), [
				ReadCommnType(name+'[i]', type),
			]),
		];
	};

	var elemTypeStr = type => typeStr(Object.assign({}, type, {arr: false}));
	var ReadAtoms = () => [
		StrStmt(`// ReadAtoms`),
		For(StrStmt(`r.N > 0`), [
			DeclVar('cc4', 'string'),
			DeclVar('ar', '*io.LimitedReader'),
			CallCheckAssign('ReadAtomHeader', ['r', '""'], ['ar', 'cc4']),
			Switch('cc4', opts.fields.map(field => [
				`"${atoms[field.type.struct].cc4}"`, [
					field.type.arr ? [
						DeclVar('item', elemTypeStr(field.type)),
						CallCheckAssign('Read'+field.type.Struct, ['ar'], ['item']),
						StrStmt(`self.${field.name} = append(self.${field.name}, item)`),
					] : [
						CallCheckAssign('Read'+field.type.Struct, ['ar'], [`self.${field.name}`]),
					],
				]
			]), showlog && [StrStmt(`log.Println("skip", cc4)`)]),
			CallCheckAssign('ReadDummy', ['ar', 'int(ar.N)'], ['_']),
		])
	];

	var ReadCommnType = (name, type) => {
		if (type.struct)
			return CallCheckAssign(
				'Read'+type.Struct, ['r'], [name]);
		return [
			//DebugStmt(type),
			CallCheckAssign(
				'Read'+type.fn, ['r', type.len].nonull(), [name]),
		]
	};

	var ReadField = (name, type) => {
		if (name == '_')
			return CallCheckAssign('ReadDummy', ['r', type.len], ['_']);
		if (type.arr && type.fn != 'Bytes')
			return ReadArr('self.'+name, type);
		return ReadCommnType('self.'+name, type);
	};

	var ReadFields = () => opts.fields.map(field => {
		var name = field.name;
		var type = field.type;
		return ReadField(name, type);
	}).nonull();

	var ptr = opts.cc4;

	return Func(
		'Read'+opts.type,
		[['r', '*io.LimitedReader']],
		[[ptr?'res':'self', (ptr?'*':'')+opts.type], ['err', 'error']],
		[ 
			ptr && StrStmt(`self := &${opts.type}{}`),
			!opts.atoms ? ReadFields() : ReadAtoms(),
			ptr && StrStmt(`res = self`),
		]
	);
};

var D = (cls, ...fields) => {
	global[cls] = (...args) => {
		var obj = {cls: cls};
		fields.forEach((k, i) => obj[k] = args[i]);
		return obj;
	};
};

D('Func', 'name', 'args', 'rets', 'body');
D('CallCheckAssign', 'fn', 'args', 'rets', 'action');
D('DeclVar', 'name', 'type');
D('For', 'cond', 'body');
D('RangeN', 'i', 'n');
D('DeclStruct', 'name', 'body');
D('StrStmt', 'content');
D('Switch', 'cond', 'cases', 'default');

var showlog = false;
var S = s => s && s || '';

var dumpFn = f => {
	var dumpArgs = x => x.map(x => x.join(' ')).join(',');
	return `func ${f.name}(${dumpArgs(f.args)}) (${dumpArgs(f.rets)}) {
		${S(showlog && 'log.Println("'+f.name+'")')}
		${dumpStmts(f.body)}
		return
	}`;
};

var dumpStmts = stmts => {
	var dumpStmt = stmt => {
		if (stmt instanceof Array) {
			return dumpStmts(stmt);
		} if (stmt.cls == 'CallCheckAssign') {
			return `if ${stmt.rets.concat(['err']).join(',')} = ${stmt.fn}(${stmt.args.join(',')}); err != nil {
				${stmt.action ? stmt.action : 'return'}
			}`;
		} else if (stmt.cls == 'DeclVar') {
			return `var ${stmt.name} ${stmt.type}`;
		} else if (stmt.cls == 'For') {
			return `for ${dumpStmt(stmt.cond)} {
				${dumpStmts(stmt.body)}
			}`;
		} else if (stmt.cls == 'RangeN') {
			return `${stmt.i} := 0; ${stmt.i} < ${stmt.n}; ${stmt.i}++`;
		} else if (stmt.cls == 'DeclStruct') {
			return `type ${stmt.name} struct {
				${stmt.body.map(line => line.join(' ')).join('\n')}
			}`;
		} else if (stmt.cls == 'Func') {
			return dumpFn(stmt);
		} else if (stmt.cls == 'StrStmt') {
			return stmt.content;
		} else if (stmt.cls == 'Switch') {
			var dumpCase = c => `case ${c[0]}: { ${dumpStmts(c[1])} }`;
			var dumpDefault = c => `default: { ${dumpStmts(c)} }`;
			return `switch ${stmt.cond} {
				${stmt.cases.map(dumpCase).join('\n')}
				${stmt.default && dumpDefault(stmt.default) || ''}
			}`;
		}
	};
	return stmts.nonull().map(dumpStmt).join('\n')
};

var parseType = s => {
	var r = {};
	var bracket = /^\[(.*)\]/;
	if (s.match(bracket)) {
		var count = s.match(bracket)[1];
		if (count.substr(0,3) == 'int') {
			r.varcount = +count.substr(3)/8;
		} else {
			r.count = +count;
		}
		r.arr = true;
		s = s.replace(bracket, '');
	}
	if (s.substr(0,1) == '*') {
		r.ptr = true;
		s = s.slice(1);
	}
	var types = /^(int|TimeStamp|byte|cc)/;
	if (s.match(types)) {
		r.type = s.match(types)[0];
		r.fn = uc(r.type);
		s = s.replace(types, '');
	}
	if (r.type == 'byte' && r.arr) {
		r.len = r.count;
		r.fn = 'Bytes';
	}
	var lenDiv = 8;
	if (r.type == 'cc') {
		r.fn = 'String';
		r.type = 'string';
		lenDiv = 1;
	}
	var number = /[0-9]+/;
	if (s.match(number)) {
		r.len = +s.match(number)[0]/lenDiv;
		s = s.replace(number, '');
	}
	if (s != '') {
		r.struct = s;
		r.Struct = uc(s);
	}
	return r;
};

var typeStr = (t) => {
	var s = '';
	if (t.arr) 
		s += '['+(t.count||'')+']';
	if (t.ptr)
		s += '*';
	if (t.struct)
		s += t.Struct;
	if (t.type)
		s += t.type;
	return s;
};

var nameShouldHide = (name) => name == '_'

var allStmts = () => {
	var stmts = [];

	for (var k in atoms) {
		var atom = atoms[k];

		var name = uc(k);
		var fields = (atom.fields || atom.atoms).map(field => {
			return {
				name: uc(field[0]),
				type: parseType(field[1]),
			};
		});

		stmts = stmts.concat([
			DeclStruct(name, fields.map(field => !nameShouldHide(field.name) && [
				uc(field.name),
				typeStr(field.type),
			]).nonull()),

			DeclReadFunc({
				type: name,
				fields: fields,
				cc4: atom.cc4,
				atoms: atom.atoms != null,
			}),
		]);
	}

	return stmts;
};

console.log(`
// THIS FILE IS AUTO GENERATED
package atom
import (
	"io"
	${showlog && '"log"' || ''}
)
`, dumpStmts(allStmts()));

