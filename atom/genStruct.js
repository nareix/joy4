
var uc = x => x && x.substr(0,1).toUpperCase()+x.slice(1);
Array.prototype.nonull = function () {
	return this.filter(x => x);
};

var Int = (name,len) => {return {name:uc(name),len,type:'int',fn:'Int'}};
var Str = (name,len) => {return {name:uc(name),len,type:'string',fn:'String'}};
var TimeStamp = (name,len) => {return {name:uc(name),len,type:'TimeStamp',fn:'TimeStamp'}};
var Bytes = (name,len) => {return {name:uc(name),len,type:'[]byte',fn:'Bytes'}};
var BytesLeft = (name) => {return {name:uc(name),type:'[]byte',fn:'BytesLeft'}};
var Fixed32 = (name,len) => {return {name:uc(name),len,type:'Fixed32',fn:'Fixed32'}};

var Atom = (type,name) => {return {name:uc(name),type:uc(type)+'Atom',fn:uc(type)+'Atom'}};
var AtomPtr = (type,name) => {return {name:uc(name),type:'*'+uc(type)+'Atom',fn:uc(type)+'Atom'}};

var Struct = (type,name) => {return {name:uc(name),type:uc(type),fn:uc(type)}};
var StructPtr = (type,name) => {return {name:uc(name),type:'*'+uc(type),fn:uc(type)}};

var Arr = (name,elem,count) => {return {name:uc(name),elem,count,type:'[]'+elem.type}};
var LenArr = (sizelen,name,elem) => {return {sizelen,name:uc(name),elem,type:'[]'+elem.type}};

var Size = (len) => {return {len,hide:true,fn:'Int'}};
var _ = (len) => {return {len,hide:true,fn:'Dummy'}};

var atoms = {
	fileType: [
		'ftyp',
		AtomPtr('movie', 'movie'),
	],

	movie: [
		'moov',
		AtomPtr('movieHeader', 'header'),
		Arr('tracks', AtomPtr('track')),
	],

	movieHeader: [
		'mvhd',
		Int('version', 1),
		Int('flags', 3),
		TimeStamp('cTime', 4),
		TimeStamp('mTime', 4),
		Int('timeScale', 4),
		Int('duration', 4),
		Int('preferredRate', 4),
		Int('preferredVolume', 2),
		_(10),
		Bytes('matrix', 36),
		TimeStamp('previewTime', 4),
		TimeStamp('previewDuration', 4),
		TimeStamp('posterTime', 4),
		TimeStamp('selectionTime', 4),
		TimeStamp('selectionDuration', 4),
		TimeStamp('currentTime', 4),
		Int('nextTrackId', 4),
	],

	track: [
		'trak',
		AtomPtr('trackHeader', 'header'),
		AtomPtr('media', 'media'),
	],

	trackHeader: [
		'tkhd',
		Int('version', 1),
		Int('flags', 3),
		TimeStamp('cTime', 4),
		TimeStamp('mTime', 4),
		Int('trackId', 4),
		_(4),
		Int('duration', 4),
		_(8),
		Int('layer', 2),
		Int('alternateGroup', 2),
		Int('volume', 2),
		_(2),
		Bytes('matrix', 36),
		Fixed32('trackWidth', 4),
		Fixed32('trackHeight', 4),
	],

	media: [
		'mdia',
		AtomPtr('mediaHeader', 'header'),
		AtomPtr('mediaInfo', 'info'),
	],

	mediaHeader: [
		'mdhd',
		Int('version', 1),
		Int('flags', 3),
		TimeStamp('cTime', 4),
		TimeStamp('mTime', 4),
		Int('timeScale', 4),
		Int('duration', 4),
		Int('language', 2),
		Int('quality', 2),
	],

	mediaInfo: [
		'minf',
		AtomPtr('videoMediaInfo', 'video'),
		AtomPtr('sampleTable', 'sample'),
	],

	videoMediaInfo: [
		'vmhd',
		Int('version', 1),
		Int('flags', 3),
		Int('graphicsMode', 2),
		Arr('opcolor', Int('', 2), 3),
	],

	sampleTable: [
		'stbl',
		AtomPtr('sampleDesc', 'sampleDesc'),
		AtomPtr('timeToSample', 'timeToSample'),
		AtomPtr('compositionOffset', 'compositionOffset'),
		AtomPtr('syncSample', 'syncSample'),
		AtomPtr('sampleSize', 'sampleSize'),
		AtomPtr('chunkOffset', 'chunkOffset'),
	],

	sampleDesc: [
		'stsd',
		Int('version', 1),
		Int('flags', 3),
		LenArr(4, 'entries', Struct('sampleDescEntry')),
	],

	timeToSample: [
		'stts',
		Int('version', 1),
		Int('flags', 3),
		LenArr(4, 'entries', Struct('timeToSampleEntry')),
	],

	compositionOffset: [
		'ctts',
		Int('version', 1),
		Int('flags', 3),
		LenArr(4, 'entries', Struct('compositionOffsetEntry')),
	],

	syncSample: [
		'stss',
		Int('version', 1),
		Int('flags', 3),
		LenArr(4, 'entries', Int('', 4)),
	],

	sampleSize: [
		'stsz',
		Int('version', 1),
		Int('flags', 3),
		LenArr(4, 'entries', Int('', 4)),
	],

	chunkOffset: [
		'stco',
		Int('version', 1),
		Int('flags', 3),
		LenArr(4, 'entries', Int('', 4)),
	],
};

var structs = {
	sampleDescEntry: [
		Size(4),
		Str('format', 4),
		_(6),
		Int('dataRefIdx', 2),
		BytesLeft('data'),
	],

	timeToSampleEntry: [
		Int('count', 4),
		Int('duration', 4),
	],

	compositionOffsetEntry: [
		Int('count', 4),
		Int('offset', 4),
	],
};

var genReadStmts = (opts) => {
	var stmts = [];

	if (opts.resIsPtr)
		stmts = stmts.concat([StrStmt(`self := &${opts.atomType}{}`)]);

	var readElemStmts = field => {
		var arr = 'self.'+field.name;
		return [
			DeclVar('item', field.elem.type),
			CallCheckAssign('Read'+field.elem.fn, ['r', field.elem.len].nonull(), ['item']),
			StrStmt(`${arr} = append(${arr}, item)`),
		]
	};

	stmts = stmts.concat(opts.fields.map(field => {
		if (field.sizelen) {
			var arr = 'self.'+field.name;
			return [
				DeclVar('count', 'int'),
				CallCheckAssign('ReadInt', ['r', field.sizelen], ['count']),
				For(RangeN('i', 'count'), readElemStmts(field)),
			];
		} else if (field.elem) {
			var cond = field.count ? RangeN('i', field.count) : StrStmt('r.N > 0');
			return For(cond, readElemStmts(field));
		} else if (!field.hide) {
			return CallCheckAssign('Read'+field.fn, ['r', field.len].nonull(), ['self.'+field.name]);
		}
	}).nonull());

	if (opts.resIsPtr)
		stmts = stmts.concat([StrStmt(`res = self`)]);

	return Func(
		'Read'+opts.fnName,
		[['r', '*io.LimitedReader']],
		[[opts.resIsPtr?'res':'self', (opts.resIsPtr?'*':'')+opts.atomType], ['err', 'error']],
		stmts
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
D('CallCheckAssign', 'fn', 'args', 'rets');
D('DeclVar', 'name', 'type');
D('For', 'cond', 'body');
D('RangeN', 'i', 'n');
D('DeclStruct', 'name', 'body');
D('StrStmt', 'content');

var dumpFn = f => {
	var dumpArgs = x => x.map(x => x.join(' ')).join(',');
	return `func ${f.name}(${dumpArgs(f.args)}) (${dumpArgs(f.rets)}) {
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
				return
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
		}
	};
	return stmts.map(dumpStmt).join('\n')
};

(() => {
	var len = 3;
	var f = Func('Readxx', [['f', '*io.LimitedReader']], [['res', '*xx'], ['err', 'error']], [
		CallCheckAssign('ReadInt', ['f', len], ['self.xx']),
		CallCheckAssign('WriteInt', ['f', len], ['self.xx']),
		DeclVar('n', 'int'),
		For(RangeN('i', 'n'), [
			CallCheckAssign('WriteInt', ['f', len], ['self.xx']),
			DeclStruct('hi', [['a', 'b'], ['c', 'd'], ['e', 'f']]),
		]),
	]);
	console.log(dumpFn(f));
});

var allStmts = () => {
	var stmts = [];

	var convStructFields = fields => {
		var typeStr = field => (
			field.cls == 'AtomPtr' || field.cls == 'StructPtr') ? '*'+field.type : field.type;

		return fields.filter(field => !field.hide)
		.map(field => {
			if (field.cls == 'Arr' || field.cls == 'LenArr')
				return [field.name, '[]'+typeStr(field.elem)];
			return [field.name, typeStr(field)];
		});
	};

	for (var k in atoms) {
		var list = atoms[k];
		var name = uc(k)+'Atom';
		var cc4 = list[0];
		var fields = list.slice(1);

		stmts = stmts.concat([
			DeclStruct(name, convStructFields(fields)),
			genReadStmts({
				cc4: cc4,
				fields: fields,
				fnName: name,
				atomType: name,
				resIsPtr: true,
			}),
		]);
	}

	for (var k in structs) {
		var fields = structs[k];
		var name = uc(k);

		stmts = stmts.concat([
			DeclStruct(name, convStructFields(fields)),
			genReadStmts({
				fields: fields,
				fnName: name,
				atomType: name,
			}),
		]);
	}

	return stmts;
};

console.log(`// THIS FILE IS AUTO GENERATED
package atom
import (
	"io"
)
`, dumpStmts(allStmts()));

