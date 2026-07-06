use criterion::{black_box, criterion_group, criterion_main, Criterion};
use era_agent::sample;
use era_agent::sanitize;
fn bench_sanitize(c: &mut Criterion) {
    c.bench_function("sanitize_process_event", |b| {
        b.iter(|| {
            let raw = sample::process_envelope("C:/Windows/System32/cmd.exe");
            black_box(sanitize::sanitize(raw, "bench-key"));
        });
    });
}

criterion_group!(benches, bench_sanitize);
criterion_main!(benches);
