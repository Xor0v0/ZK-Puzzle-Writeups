use halo2_proofs::{
    arithmetic::Field,
    circuit::{AssignedCell, Chip, Layouter, Region, Value},
    plonk::{Advice, Column, ConstraintSystem, Error, Expression, Instance, Selector, TableColumn},
    poly::Rotation,
};
use std::marker::PhantomData;

const RANGE_BITS: usize = 8;

pub struct DivChip<F: Field> {
    pub config: DivConfig,
    _marker: PhantomData<F>,
}

// You could delete or add columns here
#[derive(Clone, Debug)]
pub struct DivConfig {
    // Dividend
    a: Column<Advice>,
    // Divisor
    b: Column<Advice>,
    // Quotient
    c: Column<Advice>,
    // Remainder
    r: Column<Advice>,
    // Aux
    k: Column<Advice>,
    // Range
    range: TableColumn,
    // Instance
    instance: Column<Instance>,
    // Selector
    selector: Selector,
}

impl<F: Field> Chip<F> for DivChip<F> {
    type Config = DivConfig;
    type Loaded = ();

    fn config(&self) -> &Self::Config {
        &self.config
    }

    fn loaded(&self) -> &Self::Loaded {
        &()
    }
}

impl<F: Field> DivChip<F> {
    pub fn construct(config: <Self as Chip<F>>::Config) -> Self {
        Self {
            config,
            _marker: PhantomData,
        }
    }

    pub fn configure(meta: &mut ConstraintSystem<F>) -> <Self as Chip<F>>::Config {
        // Witness
        let col_a = meta.advice_column();
        let col_b = meta.advice_column();
        let col_c = meta.advice_column();
        let col_r = meta.advice_column();
        let col_k = meta.advice_column();
        // Selector
        let selector = meta.complex_selector();
        // Range
        let range = meta.lookup_table_column();
        // Instance
        let instance = meta.instance_column();

        meta.enable_equality(col_a);
        meta.enable_equality(col_b);
        meta.enable_equality(col_c);
        meta.enable_equality(col_r);
        meta.enable_equality(col_k);
        meta.enable_equality(instance);

        ///////////////////////// Please implement code here /////////////////////////
        meta.create_gate("div check", |meta| {
            let a = meta.query_advice(col_a, Rotation::cur());
            let b = meta.query_advice(col_b, Rotation::cur());
            let c = meta.query_advice(col_c, Rotation::cur());
            let r = meta.query_advice(col_r, Rotation::cur());
            let k = meta.query_advice(col_k, Rotation::cur());
            let s = meta.query_selector(selector);

            vec![s.clone() * (a - b.clone() * c - r.clone()), s * (b - r - k)]
        });

        meta.lookup("range check: a", |meta| {
            let s = meta.query_selector(selector);
            let a = meta.query_advice(col_a, Rotation::cur());
            vec![(s * a, range)]
        });

        meta.lookup("range check: b", |meta| {
            let s = meta.query_selector(selector);
            let b = meta.query_advice(col_b, Rotation::cur());
            vec![(s * b, range)]
        });

        meta.lookup("range check: c", |meta| {
            let s = meta.query_selector(selector);
            let c = meta.query_advice(col_c, Rotation::cur());
            vec![(s * c, range)]
        });

        meta.lookup("range check: r", |meta| {
            let s = meta.query_selector(selector);
            let r = meta.query_advice(col_r, Rotation::cur());
            vec![(s * r, range)]
        });

        meta.lookup("range check: k", |meta| {
            let s = meta.query_selector(selector);
            let k = meta.query_advice(col_k, Rotation::cur());
            vec![(s * k, range)]
        });
        ///////////////////////// End implement /////////////////////////

        DivConfig {
            a: col_a,
            b: col_b,
            c: col_c,
            r: col_r,
            k: col_k,
            range,
            instance,
            selector,
        }
    }

    // Assign range for U8 range check
    pub fn assign_range(&self, mut layouter: impl Layouter<F>) -> Result<(), Error> {
        let config = &self.config;

        ///////////////////////// Please implement code here /////////////////////////
        let mut counter = F::zero();
        layouter.assign_table(
            || "assign table_col",
            |mut table| {
                for i in 0..(1 << RANGE_BITS) {
                    table.assign_cell(
                        || "assign range",
                        config.range,
                        i,
                        || Value::known(counter),
                    )?;
                    counter += F::one();
                }
                Ok(())
            },
        );
        ///////////////////////// End implement /////////////////////////

        Ok(())
    }

    // Assign witness for division
    pub fn assign_witness(
        &self,
        mut layouter: impl Layouter<F>,
        a: F,
        b: F,
        c: F,
    ) -> Result<AssignedCell<F, F>, Error> {
        let config = self.config;

        ///////////////////////// Please implement code here /////////////////////////
        let cell_c = layouter.assign_region(
            || "assign one row of the region",
            |mut region| {
                region.assign_advice(|| "assign a", config.a, 0, || Value::known(a))?;
                region.assign_advice(|| "assign b", config.b, 0, || Value::known(b))?;
                let cell_c =
                    region.assign_advice(|| "assign c", config.c, 0, || Value::known(c))?;
                let r = a % b;
                region.assign_advice(|| "assign r", config.r, 0, || Value::known(r))?;
                region.assign_advice(|| "assign k", config.k, 0, || Value::known(b - r))?;
                config.selector.enable(&mut region, 0)?;

                Ok(cell_c)
            },
        )?;
        Ok(cell_c)
        ///////////////////////// End implement /////////////////////////
    }

    pub fn expose_public(
        &self,
        mut layouter: impl Layouter<F>,
        cell: &AssignedCell<F, F>,
    ) -> Result<(), Error> {
        layouter.constrain_instance(cell.cell(), self.config.instance, 0)
    }
}

/* ================Circuit========================== */
use halo2_proofs::circuit::SimpleFloorPlanner;
use halo2_proofs::plonk::Circuit;
#[derive(Clone, Debug)]
pub struct CircuitConfig {
    config: DivConfig,
}

#[derive(Default, Debug)]
pub struct DivCircuit<F: Field> {
    pub a: F,
    pub b: F,
    pub c: F,
}

impl<F: Field> Circuit<F> for DivCircuit<F> {
    type Config = CircuitConfig;
    type FloorPlanner = SimpleFloorPlanner;
    #[cfg(feature = "circuit-params")]
    type Params = ();

    fn without_witnesses(&self) -> Self {
        Self::default()
    }

    fn configure(meta: &mut ConstraintSystem<F>) -> Self::Config {
        let config = DivChip::<F>::configure(meta);
        CircuitConfig { config }
    }

    fn synthesize(
        &self,
        config: Self::Config,
        mut layouter: impl Layouter<F>,
    ) -> Result<(), Error> {
        let chip = DivChip::<F>::construct(config.config);
        chip.assign_range(layouter.namespace(|| "assign range"))?;
        let cell_c = chip.assign_witness(
            layouter.namespace(|| "assign witness"),
            self.a,
            self.b,
            self.c,
        )?;
        chip.expose_public(layouter.namespace(|| "expose public"), &cell_c)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use ff::PrimeField;
    use halo2_proofs::dev::MockProver;
    use halo2curves::bn256::Fr;

    #[test]
    fn sanity_check() {
        let k = 10;
        let a = Fr::from_u128(10);
        let b = Fr::from_u128(3);
        let c = Fr::from_u128(3);
        let circuit: DivCircuit<Fr> = DivCircuit { a, b, c };
        let prover = MockProver::run(k, &circuit, vec![vec![c]]).unwrap();
        assert_eq!(prover.verify(), Ok(()));
    }
}

use halo2_proofs::{
    plonk::{create_proof, keygen_pk, keygen_vk, verify_proof, ProvingKey},
    poly::kzg::{
        commitment::{KZGCommitmentScheme, ParamsKZG},
        multiopen::{ProverGWC, VerifierGWC},
        strategy::SingleStrategy,
    },
    transcript::{
        Blake2bRead, Blake2bWrite, Challenge255, TranscriptReadBuffer, TranscriptWriterBuffer,
    },
    SerdeFormat,
};
use halo2curves::bn256::{Bn256, Fr, G1Affine};
use rand::rngs::OsRng;
use std::{
    fs::File,
    io::{BufReader, BufWriter, Write},
};

fn generate_keys(k: u32, circuit: &DivCircuit<Fr>) -> (ParamsKZG<Bn256>, ProvingKey<G1Affine>) {
    let params = ParamsKZG::<Bn256>::setup(k, OsRng);
    let vk = keygen_vk(&params, circuit).expect("vk should not fail");
    let pk = keygen_pk(&params, vk, circuit).expect("pk should not fail");
    (params, pk)
}

fn generate_proof(k: u32, circuit: DivCircuit<Fr>) {
    let (params, pk) = generate_keys(k, &circuit);
    let instances: &[&[Fr]] = &[&[circuit.c]];
    let f = File::create(format!("{}", "proof")).unwrap();
    let mut proof_writer = BufWriter::new(f);
    let mut transcript = Blake2bWrite::<_, _, Challenge255<_>>::init(&mut proof_writer);
    create_proof::<
        KZGCommitmentScheme<Bn256>,
        ProverGWC<'_, Bn256>,
        Challenge255<G1Affine>,
        _,
        Blake2bWrite<_, G1Affine, Challenge255<_>>,
        _,
    >(
        &params,
        &pk,
        &[circuit],
        &[instances],
        OsRng,
        &mut transcript,
    )
    .expect("prover should not fail");
    let proof_writer = transcript.finalize();
    let _ = proof_writer.flush();
    // Dump params
    {
        let f = File::create(format!("{}", "param")).unwrap();
        let mut writer = BufWriter::new(f);
        params
            .write_custom(&mut writer, SerdeFormat::RawBytes)
            .unwrap();
        let _ = writer.flush();
    }
    // Dump vk
    {
        let f = File::create(format!("{}", "vk")).unwrap();
        let mut writer = BufWriter::new(f);
        pk.get_vk()
            .write(&mut writer, SerdeFormat::RawBytes)
            .unwrap();
        let _ = writer.flush();
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use ff::PrimeField;
    use halo2_proofs::dev::MockProver;
    use halo2curves::bn256::Fr;

    #[test]
    fn sanity_check() {
        let k = 10;
        let a = Fr::from_u128(10);
        let b = Fr::from_u128(3);
        let c = Fr::from_u128(3);
        let circuit: DivCircuit<Fr> = DivCircuit { a, b, c };
        let prover = MockProver::run(k, &circuit, vec![vec![c]]).unwrap();
        assert_eq!(prover.verify(), Ok(()));
    }
}
