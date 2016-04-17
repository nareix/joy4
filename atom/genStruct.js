
var uc = x => x && x.substr(0,1).toUpperCase()+x.slice(1);
Array.prototype.nonull = function () {
	return this.filter(x => x);
};

var atoms = {
	movie: {
		cc4: 'moov',
		fields: [
			['$atoms', [
				['header', '*movieHeader'],
				['iods', '*iods'],
				['tracks', '[]*track'],
			]],
		],
	},

	iods: {
		cc4: 'iods',
		fields: [
			['data', '[]byte'],
		],
	},

	movieHeader: {
		cc4: 'mvhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['createTime', 'TimeStamp32'],
			['modifyTime', 'TimeStamp32'],
			['timeScale', 'int32'],
			['duration', 'int32'],
			['preferredRate', 'Fixed32'],
			['preferredVolume', 'Fixed16'],
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
		fields: [
			['$atoms', [
				['header', '*trackHeader'],
				['media', '*media'],
			]],
		],
	},

	trackHeader: {
		cc4: 'tkhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['createTime', 'TimeStamp32'],
			['modifyTime', 'TimeStamp32'],
			['trackId', 'int32'],
			['_', '[4]byte'],
			['duration', 'int32'],
			['_', '[8]byte'],
			['layer', 'int16'],
			['alternateGroup', 'int16'],
			['volume', 'Fixed16'],
			['_', '[2]byte'],
			['matrix', '[9]int32'],
			['trackWidth', 'Fixed32'],
			['trackHeight', 'Fixed32'],
		],
	},

	handlerRefer: {
		cc4: 'hdlr',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['type', '[4]char'],
			['subType', '[4]char'],
			['name', '[]char'],
		],
	},

	media: {
		cc4: 'mdia',
		fields: [
			['$atoms', [
				['header', '*mediaHeader'],
				['handler', '*handlerRefer'],
				['info', '*mediaInfo'],
			]],
		],
	},

	mediaHeader: {
		cc4: 'mdhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['createTime', 'TimeStamp32'],
			['modifyTime', 'TimeStamp32'],
			['timeScale', 'int32'],
			['duration', 'int32'],
			['language', 'int16'],
			['quality', 'int16'],
		],
	},

	mediaInfo: {
		cc4: 'minf',
		fields: [
			['$atoms', [
				['sound', '*soundMediaInfo'],
				['video', '*videoMediaInfo'],
				['data', '*dataInfo'],
				['sample', '*sampleTable'],
			]],
		],
	},

	dataInfo: {
		cc4: 'dinf',
		fields: [
			['$atoms', [
				['refer', '*dataRefer'],
			]],
		],
	},

	dataRefer: {
		cc4: 'dref',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],

			['$atomsCount', 'int32'],
			['$atoms', [
				['url', '*dataReferUrl'],
			]],
		],
	},

	dataReferUrl: {
		cc4: 'url ',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
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
		fields: [
			['$atoms', [
				['sampleDesc', '*sampleDesc'],
				['timeToSample', '*timeToSample'],
				['compositionOffset', '*compositionOffset'],
				['sampleToChunk', '*sampleToChunk'],
				['syncSample', '*syncSample'],
				['chunkOffset', '*chunkOffset'],
				['sampleSize', '*sampleSize'],
			]],
		],
	},

	sampleDesc: {
		cc4: 'stsd',
		fields: [
			['version', 'int8'],
			['_', '[3]byte'],
			['$atomsCount', 'int32'],
			['$atoms', [
				['avc1Desc', '*avc1Desc'],
				['mp4aDesc', '*mp4aDesc'],
			]],
		],
	},

	mp4aDesc: {
		cc4: 'mp4a',
		fields: [
			['_', '[6]byte'],
			['dataRefIdx', 'int16'],
			['version', 'int16'],
			['revisionLevel', 'int16'],
			['vendor', 'int32'],
			['numberOfChannels', 'int16'],
			['sampleSize', 'int16'],
			['compressionId', 'int16'],
			['_', 'int16'],
			['sampleRate', 'Fixed32'],
			['$atoms', [
				['conf', '*elemStreamDesc'],
			]],
		],
	},

	elemStreamDesc: {
		cc4: 'esds',
		fields: [
			['version', 'int32'],
			['data', '[]byte'],
		],
	},

	avc1Desc: {
		cc4: 'avc1',
		fields: [
			['_', '[6]byte'],
			['dataRefIdx', 'int16'],
			['version', 'int16'],
			['revision', 'int16'],
			['vendor', 'int32'],
			['temporalQuality', 'int32'],
			['spatialQuality', 'int32'],
			['width', 'int16'],
			['height', 'int16'],
			['horizontalResolution', 'Fixed32'],
			['vorizontalResolution', 'Fixed32'],
			['_', 'int32'],
			['frameCount', 'int16'],
			['compressorName', '[32]char'],
			['depth', 'int16'],
			['colorTableId', 'int16'],

			['$atoms', [
				['conf', '*avc1Conf'],
			]],
		],
	},

	avc1Conf: {
		cc4: 'avcC',
		fields: [
			['record', 'AVCDecoderConfRecord'],
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
			['entries', '[int32]compositionOffsetEntry'],
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
			['sampleSize', 'int32'],
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

	movieFrag: {
		cc4: 'moof',
		fields: [
			['$atoms', [
				['header', '*movieFragHeader'],
				['tracks', '[]*trackFrag'],
			]],
		],
	},

	trackFragDecodeTime: {
		cc4: 'tfdt',
	},

	movieFragHeader: {
		cc4: 'mfhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['seqNum', 'int32'],
		],
	},

	trackFrag: {
		cc4: 'traf',
		fields: [
			['$atoms', [
				['header', '*trackFragHeader'],
				['decodeTime', '*trackFragDecodeTime'],
				['run', '*trackFragRun'],
			]],
		],
	},

	trackFragRun: {
		cc4: 'trun',
	},

	trackFragHeader: {
		cc4: 'tfhd',
	},

	/*
	// need hand write
	trackFragRun: {
		cc4: 'trun',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['sampleCount', 'int32'],
			['dataOffset', 'int32'],
			['entries', '[]int32'],
		],
	},

	trackFragHeader: {
		cc4: 'tfhd',
		fields: [
			['version', 'int8'],
			['flags', 'int24'],
			['id', 'int32'],
			['sampleDescriptionIndex', 'int32'],
			['_', '[12]byte'],
		],
	},
	*/
};

var DeclReadFunc = (opts) => {
	var stmts = [];

	var DebugStmt = type => `// ${JSON.stringify(type)}`;

	var ReadArr = (name, type) => {
		return [
			//StrStmt('// ReadArr'),
			//DebugStmt(type),
			type.varcount && [
				DeclVar('count', 'int'),
				CallCheckAssign('ReadInt', ['r', type.varcount], ['count']),
				`${name} = make(${typeStr(type)}, count)`,
			],
			For(RangeN('i', type.varcount ? 'count' : type.count), [
				ReadCommnType(name+'[i]', type),
			]),
		];
	};

	var elemTypeStr = type => typeStr(Object.assign({}, type, {arr: false}));
	var ReadAtoms = fields => [
		For(`r.N > 0`, [
			DeclVar('cc4', 'string'),
			DeclVar('ar', '*io.LimitedReader'),
			CallCheckAssign('ReadAtomHeader', ['r', '""'], ['ar', 'cc4']),
			Switch('cc4', fields.map(field => [
				`"${atoms[field.type.struct].cc4}"`, [
					field.type.arr ? [
						DeclVar('item', elemTypeStr(field.type)),
						CallCheckAssign('Read'+field.type.Struct, ['ar'], ['item']),
						`self.${field.name} = append(self.${field.name}, item)`,
					] : [
						CallCheckAssign('Read'+field.type.Struct, ['ar'], [`self.${field.name}`]),
					],
				]
			]), showlog && [`log.Println("skip", cc4)`]),
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
				'Read'+type.fn, ['r', type.len||'int(r.N)'], [name]),
		]
	};

	var ReadField = (name, type) => {
		if (name == '_')
			return CallCheckAssign('ReadDummy', ['r', type.len], ['_']);
		if (name == '$atoms')
			return ReadAtoms(type.list);
		if (name == '$atomsCount')
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
			ptr && `self := &${opts.type}{}`,
			ReadFields(),
			ptr && `res = self`,
		]
	);
};

var DeclWriteFunc = (opts) => {
	var SavePos = [
		DeclVar('aw', '*Writer'),
		CallCheckAssign('WriteAtomHeader', ['w', `"${opts.cc4}"`], ['aw']),
		`w = aw`,
	];

	var RestorePosSetSize = [
		CallCheckAssign('aw.Close', [], []),
	];

	var WriteAtoms = fields => fields.map(field => {
		var name = 'self.'+field.name;
		return [
			`if ${name} != nil {`,
			field.type.arr ? WriteArr(name, field.type) : WriteCommnType(name, field.type),
			atomsCount && `${atomsCount.name}++`,
			`}`,
		];
	});

	var WriteArr = (name, type) => {
		return [
			type.varcount && CallCheckAssign('WriteInt', ['w', `len(${name})`, type.varcount], []),
			For(`_, elem := range ${name}`, [
				WriteCommnType('elem', type),
			]),
		];
	};

	var WriteCommnType = (name, type) => {
		if (type.struct)
			return CallCheckAssign(
				'Write'+type.Struct, ['w', name], []);
		return [
			CallCheckAssign(
				'Write'+type.fn, ['w', name, type.len||`len(${name})`], []),
		]
	};

	var atomsCount;

	var WriteAtomsCountStart = (type) => {
		atomsCount = {
			name: 'atomsCount',
			namePos: 'atomsCountPos',
			type: type,
		}
		return [
			DeclVar(atomsCount.name, 'int'),
			DeclVar(atomsCount.namePos, 'int64'),
			CallCheckAssign('WriteEmptyInt', ['w', type.len], [atomsCount.namePos]),
		];
	};

	var WriteAtomsCountEnd = (type) => {
		return [
			CallCheckAssign('RefillInt', 
				['w', atomsCount.namePos, atomsCount.name, atomsCount.type.len],
				[]
			),
		];
	};

	var WriteField = (name, type) => {
		if (name == '_')
			return CallCheckAssign('WriteDummy', ['w', type.len], []);
		if (name == '$atoms')
			return WriteAtoms(type.list);
		if (name == '$atomsCount')
			return WriteAtomsCountStart(type);
		if (type.arr && type.fn != 'Bytes')
			return WriteArr('self.'+name, type);
		return WriteCommnType('self.'+name, type);
	};

	var WriteFields = () => opts.fields
		.map(field => WriteField(field.name, field.type))
		.concat(atomsCount && WriteAtomsCountEnd())

	return Func(
		'Write'+opts.type,
		[['w', 'io.WriteSeeker'], ['self', (opts.cc4?'*':'')+opts.type]],
		[['err', 'error']],
		[
			opts.cc4 && SavePos,
			WriteFields(),
			opts.cc4 && RestorePosSetSize,
		]
	);
};

var DeclDumpFunc = (opts) => {
	var dumpStruct = (name, type) => {
		if (type.ptr)
			return If(`${name} != nil`, Call('Walk'+type.Struct, ['w', name]));
		return Call('Walk'+type.Struct, ['w', name]);
	};

	var dumpArr = (name, type, id) => {
		return [
			//Call('w.StartArray', [`"${id}"`, `len(${name})`]),
			For(`i, item := range(${name})`, If(
				`w.FilterArrayItem("${opts.type}", "${id}", i, len(${name}))`,
				dumpCommonType('item', type, id),
				[`w.ArrayLeft(i, len(${name}))`, 'break']
			)),
			//Call('w.EndArray', []),
		];
	};

	var dumpCommonType = (name, type, id) => {
		if (type.struct)
			return dumpStruct(name, type);
		return [
			Call('w.Name', [`"${id}"`]),
			Call('w.'+type.fn, [name]),
		];
	};

	var dumpField = (name, type, noarr) => {
		if (name == '_')
			return;
		if (name == '$atomsCount')
			return;
		if (name == '$atoms') {
			return type.list.map(field => dumpField(field.name, field.type));
		}
		if (!noarr && type.arr && type.fn != 'Bytes')
			return dumpArr('self.'+name, type, name);
		return dumpCommonType('self.'+name, type, name);
	};

	var dumpFields = fields => 
		[ Call('w.StartStruct', [`"${opts.type}"`]) ]
		.concat(fields.map(field => dumpField(field.name, field.type)))
		.concat([Call('w.EndStruct', [])]);

	return Func(
		'Walk'+opts.type,
		[['w', 'Walker'], ['self', (opts.cc4?'*':'')+opts.type]],
		[],
		dumpFields(opts.fields)
	)
};

var D = (cls, ...fields) => {
	global[cls] = (...args) => {
		var obj = {cls: cls};
		fields.forEach((k, i) => obj[k] = args[i]);
		return obj;
	};
};

D('Func', 'name', 'args', 'rets', 'body');
D('If', 'cond', 'action', 'else');
D('Call', 'fn', 'args');
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

var dumpStmts = stmt => {
	if (typeof(stmt) == 'string') {
		return stmt;
	} else if (stmt instanceof Array) {
		return stmt.nonull().map(dumpStmts).join('\n');
	} else if (stmt.cls == 'If') {
		var s = `if ${stmt.cond} {
			${dumpStmts(stmt.action)}
		}`;
		if (stmt.else) {
			s += ` else {
				${dumpStmts(stmt.else)}
			}`;
		}
		return s;
	} else if (stmt.cls == 'Call') {
		return `${stmt.fn}(${stmt.args.join(',')})`;
	} else if (stmt.cls == 'CallCheckAssign') {
		return `if ${stmt.rets.concat(['err']).join(',')} = ${stmt.fn}(${stmt.args.join(',')}); err != nil {
			${stmt.action ? stmt.action : 'return'}
		}`;
	} else if (stmt.cls == 'DeclVar') {
		return `var ${stmt.name} ${stmt.type}`;
	} else if (stmt.cls == 'For') {
		return `for ${dumpStmts(stmt.cond)} {
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

var parseType = s => {
	var r = {};
	var bracket = /^\[(.*)\]/;
	var lenDiv = 8;
	var types = /^(int|TimeStamp|byte|Fixed|char)/;
	var number = /^[0-9]+/;

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

	if (s.match(types)) {
		r.type = s.match(types)[0];
		r.fn = uc(r.type);
		s = s.replace(types, '');
	}

	if (r.type == 'byte' && r.arr) {
		r.len = r.count;
		r.fn = 'Bytes';
	}

	if (r.type == 'char' && r.arr) {
		r.len = r.count;
		r.fn = 'String';
		r.type = 'string';
		r.arr = false;
		lenDiv = 1;
	}

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

	var parseFields = fields => fields.map(field => {
		return {
			name: uc(field[0]),
			type: field[0] == '$atoms' ? {list: parseFields(field[1])} : parseType(field[1]),
		};
	});

	var genStructFields = fields => fields.map(field => {
		if (field.name == '_')
			return;
		if (field.name == '$atomsCount')
			return;
		if (field.name == '$atoms')
			return field.type.list;
		return [field];
	}).nonull().reduce((prev, cur) => prev.concat(cur)).map(field => [
		field.name, typeStr(field.type)]);

	for (var k in atoms) {
		var atom = atoms[k];
		var name = uc(k);

		if (atom.fields == null)
			continue;

		var fields = parseFields(atom.fields);

		stmts = stmts.concat([
			DeclStruct(name, genStructFields(fields)),

			DeclReadFunc({
				type: name,
				fields: fields,
				cc4: atom.cc4,
			}),

			DeclWriteFunc({
				type: name,
				fields: fields,
				cc4: atom.cc4,
			}),

			DeclDumpFunc({
				type: name,
				fields: fields,
				cc4: atom.cc4,
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

