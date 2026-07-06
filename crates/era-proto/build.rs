//! Codegen из единого источника proto/era/v1/ (ADR-0001, ADR-0003, S1-1).

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_root = "../../proto";

    // Bundled protoc + well-known types — воспроизводимо в CI/air-gap.
    std::env::set_var("PROTOC", protoc_bin_vendored::protoc_bin_path()?);

    println!("cargo:rerun-if-changed={proto_root}/era/v1/envelope.proto");
    println!("cargo:rerun-if-changed={proto_root}/era/v1/ingest.proto");

    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .compile_protos(
            &[format!("{proto_root}/era/v1/ingest.proto")],
            &[proto_root],
        )?;

    Ok(())
}
