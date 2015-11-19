
var ucfirst = x => x.substr(0,1).toUpperCase() + x.slice(1);

var D = (cls, prop, ...fields) => {
	var ctor = function (args) {
		this.cls = cls;
		Object.assign(this, prop);
		fields.forEach((f, i) => this[f] = typeof(args[i]) == 'string' ? ucfirst(args[i]) : args[i]);
		if (cls == 'Atom' || cls == 'AtomPtr')
			this.type = this.type + 'Atom';
	};
	global[cls] = (...args) => new ctor(args);
}

D('Int', {basic: true, type: 'int'}, 'name', 'len');
D('Str', {basic: true, type: 'string'}, 'name', 'len');
D('TimeStamp', {basic: true, type: 'TimeStamp'}, 'name', 'len');
D('Bytes', {basic: true, type: '[]byte'}, 'name', 'len');
D('Fixed32', {basic: true, type: 'Fixed32'}, 'name', 'len');

D('Atom', {}, 'type', 'name');
D('AtomPtr', {}, 'type', 'name');
D('Struct', {}, 'type', 'name');
D('StructPtr', {}, 'type', 'name');

D('Arr', {}, 'name', 'elem', 'count');
D('LenArr', {}, 'len', 'name', 'elem', 'isptr');

D('Size', {hide: true}, 'len');
D('_', {type: 'Dummy', hide: true}, 'len');

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
		Bytes('data'),
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

var typeStr = field => (
	field.cls == 'AtomPtr' || field.cls == 'StructPtr') ? '*'+field.type : field.type;

var dumpStruct = (name, list) => {
	console.log(`type ${name} struct {
		%s
	}`, list
		.filter(field => !field.hide)
		.map(field => {
			if (field.cls == 'Arr' || field.cls == 'LenArr')
				return field.name +' []'+typeStr(field.elem);
			return field.name+' '+typeStr(field)
		}).join('\n')
	);

	dumpReadFn(name, null, false, list);
};

var dumpReadFn = (type, cc4, retptr, list) => {
	var useSize;

	if (retptr) {
		console.log(`func Read${type}(r io.LimitedReader) (res *${type}, err error) {`);
		console.log(`self := &${type}{}`);
	} else {
		console.log(`func Read${type}(r io.LimitedReader) (self ${type}, err error) {`);
	}

	list.forEach(field => {
		if (field.cls == 'Size') {
			useSize = true;
			console.log(`size := ReadInt(r, ${field.len})`);
		} else if (field.cls == 'Arr') {
			var cond = field.count ? `i := 0; i < ${field.count}; i++` : `r.N > 0`;
			console.log(`for ${cond} {`);
			console.log(`var item ${typeStr(field.elem)}`);
			console.log(`if item, err = Read${field.elem.type}(r); err != nil {
				return
			} else {
				self.${field.name} = append(self.${field.name}, item)
			}`);
			console.log(`}`);
		} else if (field.cls == 'LenArr') {
			console.log(`if n, err := ReadInt(r, ${field.len}); err != nil {
				return nil, err
			} else {
				for i := 0; i < n; i++ {
					self.${field.name} = append(self.${field.name}, item)
				}
			}`);
		} else {
			var fn = field.basic ? field.cls : field.type;
			var args = field.basic ? 'r, ' + field.len : 'r';
			console.log(`if self.${field.name}, err = Read${fn}(${args}); err != nil {
				return
			}`);
		}
	});

	if (retptr) {
		console.log(`res = self`);
	}
	console.log(`return`);
	console.log(`}`);
};

var dumpWriteFn = (name, list) => {
	var useSize;

	console.log(`func Write${name}(w Writer, atom ${name}) (err error) {`);

	list.map(x => {
		if (x.type == 'size') {
			useSize = true;
			return `ReadInt(r, ${x.len})`;
		}
	});

	console.log(`}`);
};

console.log('// THIS FILE IS AUTO GENERATED');
console.log('');
console.log('package atom');

for (var k in atoms) {
	var list = atoms[k];
	var name = ucfirst(k)+'Atom';

	var cc4 = list[0];
	list = list.slice(1);
	dumpStruct(name, list);
	dumpReadFn(name, cc4, true, list);
}

for (var k in structs) {
	var list = structs[k];
	dumpStruct(ucfirst(k), list)
}

