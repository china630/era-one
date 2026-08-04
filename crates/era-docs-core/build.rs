fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_root = "../../proto";
    println!("cargo:rerun-if-changed={proto_root}/era/v1/office.proto");
    prost_build::compile_protos(&[format!("{proto_root}/era/v1/office.proto")], &[proto_root])?;
    Ok(())
}
