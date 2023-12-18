package indexer

func DefaultExcludeDirFilters() []string {
	return []string{
		"po",

		// VCS
		"CVS",
		".svn",
		".git",
		"_darcs",
		".bzr",
		".hg",

		// development
		"CMakeFiles",
		"CMakeTmp",
		"CMakeTmpQmake",
		".moc",
		".obj",
		".pch",
		".uic",
		".npm",
		".yarn",
		".yarn-cache",
		"__pycache__",
		"node_modules",
		"node_packages",
		"nbproject",
		".terraform",
		".venv",
		"venv",
		".rbenv",
		".bundle",
		".conda",
		".cargo",
		".vscode",

		// misc
		"core-dumps",
		"lost+found",
		"drive_c", // wine giant dirs
		".wine",
		".mozilla",
		".thunderbird",

		// cache dirs
		".cache",
		"CachedData",
		"CacheStorage",
		"Cache_Data",
		"Code Cache",
		"ScriptCache",

		"store.knot",

		// do not use in production
		".local",
		".config",
	}
}
func DefaultExcludeFileFilters() []string {
	return []string{
		"*~",
		"*.part",

		// temporary build files
		"*.o",
		"*.la",
		"*.lo",
		"*.loT",
		"*.moc",
		"moc_*.cpp",
		"qrc_*.cpp",
		"ui_*.h",
		"cmake_install.cmake",
		"CMakeCache.txt",
		"CTestTestfile.cmake",
		"libtool",
		"config.status",
		"confdefs.h",
		"autom4te",
		"conftest",
		"confstat",
		"Makefile.am",
		"*.gcode", // CNC machine/3D printer toolpath files
		".ninja_deps",
		".ninja_log",
		"build.ninja",

		// misc
		"*.csproj",
		"*.m4",
		"*.rej",
		"*.gmo",
		"*.pc",
		"*.omf",
		"*.aux",
		"*.tmp",
		"*.po",
		"*.vm*",
		"*.nvram",
		"*.rcore",
		"*.swp",
		"*.swap",
		"lzo",
		"litmain.sh",
		"*.orig",
		".histfile.*",
		".xsession-errors*",
		"*.map",
		"*.so",
		"*.a",
		"*.db",
		"*.qrc",
		"*.ini",
		"*.init",
		"*.img",      // typical extension for raw disk images
		"*.vdi",      // Virtualbox disk images
		"*.vbox*",    // Virtualbox VM files
		"vbox.log",   // Virtualbox log files
		"*.qcow2",    // QEMU QCOW2 disk images
		"*.vmdk",     // VMware disk images
		"*.vhd",      // Hyper-V disk images
		"*.vhdx",     // Hyper-V disk images
		"*.sql",      // SQL database dumps
		"*.sql.gz",   // Compressed SQL database dumps
		"*.ytdl",     // youtube-dl temp files
		"*.tfstate*", // Terraform state files

		// Bytecode files
		"*.class", // Java
		"*.pyc",   // Python
		"*.pyo",   // More Python
		"*.elc",   // Emacs Lisp
		"*.qmlc",  // QML
		"*.jsc",   // Javascript

		// files known in bioinformatics containing huge amount of unindexable data
		"*.fastq",
		"*.fq",
		"*.gb",
		"*.fasta",
		"*.fna",
		"*.gbff",
		"*.faa",
		"*.fna",
	}
}
