use std::path::PathBuf;

fn main() {
    // Copy product SPAs + office-shell into assets/ for release ServeDir.
    let manifest = PathBuf::from(std::env::var("CARGO_MANIFEST_DIR").unwrap());
    let repo_ui = manifest.join("../../../ui");
    let out = manifest.join("assets");
    let _ = std::fs::create_dir_all(&out);
    copy_dir_if_exists(repo_ui.join("docs/web"), out.join("docs-web"));
    copy_dir_if_exists(repo_ui.join("tables/web"), out.join("tables-web"));
    copy_dir_if_exists(
        repo_ui.join("presentations/web"),
        out.join("presentations-web"),
    );
    copy_dir_if_exists(repo_ui.join("projects/web"), out.join("projects-web"));
    copy_dir_if_exists(repo_ui.join("office-shell/web"), out.join("office-assets"));
    copy_dir_if_exists(repo_ui.join("drive/web"), out.join("drive-web"));
    copy_dir_if_exists(repo_ui.join("office-ai/web"), out.join("office-ai-web"));
    println!("cargo:rerun-if-changed=assets/solo-docs-boot.js");
    println!("cargo:rerun-if-changed=assets/solo-docs-skin.css");
    println!("cargo:rerun-if-changed=../../../ui/docs/web");
    println!("cargo:rerun-if-changed=../../../ui/tables/web");
    println!("cargo:rerun-if-changed=../../../ui/presentations/web");
    println!("cargo:rerun-if-changed=../../../ui/projects/web");
    println!("cargo:rerun-if-changed=../../../ui/office-shell/web");
    println!("cargo:rerun-if-changed=../../../ui/shared-tokens");
    tauri_build::build();
}

fn copy_dir_if_exists(src: PathBuf, dst: PathBuf) {
    if !src.is_dir() {
        return;
    }
    let _ = std::fs::create_dir_all(&dst);
    if let Ok(entries) = std::fs::read_dir(&src) {
        for e in entries.flatten() {
            let p = e.path();
            let name = e.file_name();
            let target = dst.join(&name);
            if p.is_dir() {
                copy_dir_if_exists(p, target);
            } else if let Ok(meta) = e.metadata() {
                if meta.is_file() {
                    let _ = std::fs::copy(&p, &target);
                }
            }
        }
    }
}
