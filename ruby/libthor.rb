require "ffi"

module Libthor
    extend FFI::Library
	ffi_lib "libthor.so"
    class Setting < FFI::Struct
        layout(
			:file, :string,
			:profile, :string,
			:kmsKeyID, :string,
			:region, :string,
			:externalPath, :string,
        )
    end

    class Result < FFI::Struct
		layout(
			:msg, :string,
			:error, :string,
		)
    end

    attach_function :deploy, [:pointer], Result
end
