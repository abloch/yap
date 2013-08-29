package Application

import (
	"chukuparser/Algorithm/Model/Perceptron"
	"chukuparser/Algorithm/Transition"
	"chukuparser/NLP/Format/Conll"
	"chukuparser/NLP/Format/Lattice"
	"chukuparser/NLP/Format/Segmentation"
	"chukuparser/NLP/Parser/Dependency"
	. "chukuparser/NLP/Parser/Dependency/Transition"
	"chukuparser/NLP/Parser/Dependency/Transition/Morph"
	NLP "chukuparser/NLP/Types"
	// "chukuparser/Util"

	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var (
	RICH_FEATURES []string = []string{
		"S0|w|p", "S0|w", "S0|p", "N0|w|p",
		"N0|w", "N0|p", "N1|w|p", "N1|w",
		"N1|p", "N2|w|p", "N2|w", "N2|p",
		"S0|w|p+N0|w|p", "S0|w|p+N0|w",
		"S0|w+N0|w|p", "S0|w|p+N0|p",
		"S0|p+N0|w|p", "S0|w+N0|w",
		"S0|p+N0|p", "N0|p+N1|p",
		"N0|p+N1|p+N2|p", "S0|p+N0|p+N1|p",
		"S0h|p+S0|p+N0|p", "S0|p+S0l|p+N0|p",
		"S0|p+S0r|p+N0|p", "S0|p+N0|p+N0l|p",
		"S0|w|d", "S0|p|d", "N0|w|d", "N0|p|d",
		"S0|w+N0|w|d", "S0|p+N0|p|d",
		"S0|w|vr", "S0|p|vr", "S0|w|vl", "S0|p|vl", "N0|w|vl", "N0|p|vl",
		"S0h|w", "S0h|p", "S0|l", "S0l|w",
		"S0l|p", "S0l|l", "S0r|w", "S0r|p",
		"S0r|l", "N0l|w", "N0l|p", "N0l|l",
		"S0h2|w", "S0h2|p", "S0h|l", "S0l2|w",
		"S0l2|p", "S0l2|l", "S0r2|w", "S0r2|p",
		"S0r2|l", "N0l2|w", "N0l2|p", "N0l2|l",
		"S0|p+S0l|p+S0l2|p", "S0|p+S0r|p+S0r2|p",
		"S0|p+S0h|p+S0h2|p", "N0|p+N0l|p+N0l2|p",
		"S0|w|sr", "S0|p|sr", "S0|w|sl", "S0|p|sl",
		"N0|w|sl", "N0|p|sl",
		"N0|t",                                 // all pos tags of morph queue
		"A0|g", "A0|p", "A0|n", "A0|t", "A0|o", // agreement
		"M0|w", "M1|w", "M2|w", // lattice bigram and trigram
		"M0|w+M1|w", "M0|w+M1|w+M2|w", // bi/tri gram combined
	}

	LABELS []string = []string{
		"advmod", "amod", "appos", "aux",
		"cc", "ccomp", "comp", "complmn",
		"compound", "conj", "cop", "def",
		"dep", "det", "detmod", "gen",
		"ghd", "gobj", "hd", "mod",
		"mwe", "neg", "nn", "null",
		"num", "number", "obj", "parataxis",
		"pcomp", "pobj", "posspmod", "prd",
		"prep", "prepmod", "punct", "qaux",
		"rcmod", "rel", "relcomp", "subj",
		"tmod", "xcomp",
	}

	Iterations               int
	BeamSize                 int
	tConll, tLatDis, tLatAmb string
	tSeg                     string
	input                    string
	outLat, outSeg           string
	modelFile                string
)

// tConll := "train4k.hebtb.gold.conll"
// tLatDis := "train4k.hebtb.gold.lattices"
// tLatAmb := "train4k.hebtb.pred.lattices"
// input := "dev.hebtb.pred.conll.tobeparsed.pred_tagged+pred_token.nodisamb.lattices"
// outLat := "dev.hebtb.pred.conll"
// outSeg := "dev.hebtb.pred.segmentation"
// tSeg := "train4k.hebtb.gold.segmentation"
// tConll := "dev.hebtb.gold.conll"
// tLatDis := "dev.hebtb.gold.conll.tobeparsed.gold_tagged+gold_fixed_token.lattices"
// tLatAmb := "dev.hebtb.pred.conll.tobeparsed.pred_tagged+pred_token.nodisamb.lattices"
// input := "dev.hebtb.pred.conll.tobeparsed.pred_tagged+pred_token.nodisamb.lattices"
// outLat := "dev.hebtb.pred.conll"
// outSeg := "dev.hebtb.pred.segmentation"
// tSeg := "dev.hebtb.gold.segmentation"
// tConll := "dev.hebtb.1.gold.conll"
// tLatDis := "dev.hebtb.1.gold.conll.tobeparsed.gold_tagged+gold_fixed_token.lattices"
// tLatAmb := "dev.hebtb.1.pred.conll.tobeparsed.pred_tagged+pred_token.nodisamb.lattices"
// input := "dev.hebtb.1.pred.conll.tobeparsed.pred_tagged+pred_token.nodisamb.lattices"
// outLat := "dev.hebtb.1.pred.conll"
// outSeg := "dev.hebtb.1.pred.segmentation"
// tSeg := "dev.hebtb.1.gold.segmentation"

func TrainingSequences(trainingSet []*Morph.BasicMorphGraph, features []string) []Perceptron.DecodedInstance {
	extractor := new(GenericExtractor)
	// verify feature load
	for _, feature := range features {
		if err := extractor.LoadFeature(feature); err != nil {
			log.Panicln("Failed to load feature", err.Error())
		}
	}
	arcSystem := &Morph.ArcEagerMorph{}
	arcSystem.Relations = LABELS
	arcSystem.AddDefaultOracle()

	idleSystem := &Morph.Idle{arcSystem}
	transitionSystem := Transition.TransitionSystem(idleSystem)
	deterministic := &Deterministic{transitionSystem, extractor, false, true, false, &Morph.MorphConfiguration{}}

	decoder := Perceptron.EarlyUpdateInstanceDecoder(deterministic)
	updater := new(Perceptron.AveragedStrategy)

	perceptron := &Perceptron.LinearPerceptron{Decoder: decoder, Updater: updater}
	perceptron.Init()
	tempModel := Dependency.ParameterModel(&PerceptronModel{perceptron})

	instances := make([]Perceptron.DecodedInstance, 0, len(trainingSet))
	for i, graph := range trainingSet {
		if i%100 == 0 {
			log.Println("At line", i)
		}
		sent := graph.Lattice
		// log.Println("Gold parsing graph (nodes, arcs, lattice)")
		// log.Println("Nodes:")
		// for _, node := range graph.Nodes {
		// 	log.Println("\t", node)
		// }
		// log.Println("Arcs:")
		// for _, arc := range graph.Arcs {
		// 	log.Println("\t", arc)
		// }
		// log.Println("Mappings:")
		// for _, m := range graph.Mappings {
		// 	log.Println("\t", m)
		// }
		// log.Println("Lattices:")
		// for _, lat := range graph.Lattice {
		// 	log.Println("\t", lat)
		// }
		_, goldParams := deterministic.ParseOracle(graph, nil, tempModel)
		if goldParams != nil {
			seq := goldParams.(*ParseResultParameters).Sequence
			// log.Println("Gold seq:\n", seq)
			decoded := &Perceptron.Decoded{sent, seq[0]}
			instances = append(instances, decoded)
		}
	}
	return instances
}

func ReadTraining(filename string) []Perceptron.DecodedInstance {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	var instances []Perceptron.DecodedInstance
	dec := gob.NewDecoder(file)
	err = dec.Decode(&instances)
	if err != nil {
		panic(err)
	}
	return instances
}

func WriteTraining(instances []Perceptron.DecodedInstance, filename string) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(file)
	err = enc.Encode(instances)
	if err != nil {
		panic(err)
	}
}

func Train(trainingSet []Perceptron.DecodedInstance, Iterations, BeamSize int, features []string, filename string) *Perceptron.LinearPerceptron {
	extractor := new(GenericExtractor)
	// verify feature load
	for _, feature := range features {
		if err := extractor.LoadFeature(feature); err != nil {
			log.Panicln("Failed to load feature", err.Error())
		}
	}
	arcSystem := &Morph.ArcEagerMorph{}
	arcSystem.Relations = LABELS
	arcSystem.AddDefaultOracle()

	idleSystem := &Morph.Idle{arcSystem}
	transitionSystem := Transition.TransitionSystem(idleSystem)
	conf := &Morph.MorphConfiguration{}

	beam := Beam{
		TransFunc:      transitionSystem,
		FeatExtractor:  extractor,
		Base:           conf,
		NumRelations:   len(arcSystem.Relations),
		Size:           BeamSize,
		ConcurrentExec: true}
	varbeam := &VarBeam{beam}
	decoder := Perceptron.EarlyUpdateInstanceDecoder(varbeam)
	updater := new(Perceptron.AveragedStrategy)

	perceptron := &Perceptron.LinearPerceptron{
		Decoder:   decoder,
		Updater:   updater,
		Tempfile:  filename,
		TempLines: 1000}

	perceptron.Iterations = Iterations
	perceptron.Init()
	// perceptron.TempLoad("model.b64.i1")
	perceptron.Log = true

	perceptron.Train(trainingSet)

	return perceptron
}

func Parse(sents []NLP.LatticeSentence, BeamSize int, model Dependency.ParameterModel, features []string) []NLP.MorphDependencyGraph {
	extractor := new(GenericExtractor)
	// verify load
	for _, feature := range features {
		if err := extractor.LoadFeature(feature); err != nil {
			log.Panicln("Failed to load feature", err.Error())
		}
	}
	arcSystem := &Morph.ArcEagerMorph{}
	arcSystem.Relations = LABELS
	arcSystem.AddDefaultOracle()
	transitionSystem := Transition.TransitionSystem(arcSystem)

	conf := &Morph.MorphConfiguration{}

	beam := Beam{
		TransFunc:       transitionSystem,
		FeatExtractor:   extractor,
		Base:            conf,
		Size:            BeamSize,
		NumRelations:    len(arcSystem.Relations),
		Model:           model,
		ConcurrentExec:  true,
		ShortTempAgenda: true}

	varbeam := &VarBeam{beam}

	parsedGraphs := make([]NLP.MorphDependencyGraph, len(sents))
	for i, sent := range sents {
		log.Println("Parsing sent", i)
		graph, _ := varbeam.Parse(sent, nil, model)
		labeled := graph.(NLP.MorphDependencyGraph)
		parsedGraphs[i] = labeled
	}
	return parsedGraphs
}

func WriteModel(model Perceptron.Model, filename string) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	model.Write(file)
}

func ReadModel(filename string) *Perceptron.LinearPerceptron {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	model := new(Perceptron.LinearPerceptron)
	model.Read(file)
	return model
}

func RegisterTypes() {
	gob.Register(Transition.ConfigurationSequence{})
	gob.Register(&Morph.BasicMorphGraph{})
	gob.Register(&NLP.Morpheme{})
	gob.Register(&BasicDepArc{})
	gob.Register(&Beam{})
	gob.Register(&Morph.MorphConfiguration{})
	gob.Register(&Morph.ArcEagerMorph{})
	gob.Register(&GenericExtractor{})
	gob.Register(&PerceptronModel{})
	gob.Register(&Perceptron.AveragedStrategy{})
	gob.Register(&Perceptron.Decoded{})
	gob.Register(NLP.LatticeSentence{})
	gob.Register(&StackArray{})
	gob.Register(&ArcSetSimple{})
}

func CombineTrainingInputs(graphs []NLP.LabeledDependencyGraph, goldLats, ambLats []NLP.LatticeSentence) ([]*Morph.BasicMorphGraph, int) {
	if len(graphs) != len(goldLats) || len(graphs) != len(ambLats) {
		panic(fmt.Sprintf("Got mismatched training slice inputs (graphs, gold lattices, ambiguous lattices):", len(graphs), len(goldLats), len(ambLats)))
	}
	morphGraphs := make([]*Morph.BasicMorphGraph, len(graphs))
	var (
		numLatticeNoGold int
		noGold           bool
	)
	prefix := log.Prefix()
	for i, goldGraph := range graphs {
		goldLat := goldLats[i]
		ambLat := ambLats[i]
		log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
		morphGraphs[i], noGold = Morph.CombineToGoldMorph(goldGraph, goldLat, ambLat)
		if noGold {
			numLatticeNoGold++
		}
	}
	log.SetPrefix(prefix)
	return morphGraphs, numLatticeNoGold
}

func MorphTrainAndParse(cmd *commander.Command, args []string) {

	RegisterTypes()
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	log.Println("Configuration")
	log.Println("CPUs:             ", CPUs)
	log.Println("Beam:              Variable Length")
	log.Println("Transition System: IDLE+Morph+ArcEager")
	log.Println("Features:          Rich + Q Tags + Morph + Agreement")
	log.Println("Iterations:\t", Iterations)
	log.Println("Beam Size:\t", BeamSize)
	log.Println("Model file:\t", modelFile)
	log.Println()
	log.Println("Data")
	log.Println("Train file (conll):\t\t", tConll)
	log.Println("Train file (disamb. lattice):\t", tLatDis)
	log.Println("Train file (ambig.  lattice):\t", tLatAmb)
	log.Println("Test file  (ambig.  lattice):\t", input)
	log.Println()
	log.Println("Out (disamb.) file:\t", outLat)
	log.Println("Out (segmt.) file:\t", outSeg)
	log.Println()
	log.Println("Profiler interface:", "http://127.0.0.1:6060/debug/pprof")
	log.Println()
	// launch net server for profiling
	go func() {
		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	log.Println("Reading training conll sentences from", tConll)
	s, e := Conll.ReadFile(tConll)
	if e != nil {
		log.Println(e)
		return
	}
	log.Println("Read", len(s), "sentences from", tConll)
	log.Println("Converting from conll to internal structure")
	goldConll := Conll.Conll2GraphCorpus(s)

	log.Println("Reading training disambiguated lattices from", tLatDis)
	lDis, lDisE := Lattice.ReadFile(tLatDis)
	if lDisE != nil {
		log.Println(lDisE)
		return
	}
	log.Println("Read", len(lDis), "disambiguated lattices from", tLatDis)
	log.Println("Converting lattice format to internal structure")
	goldDisLat := Lattice.Lattice2SentenceCorpus(lDis)

	log.Println("Reading ambiguous lattices from", input)
	lAmb, lAmbE := Lattice.ReadFile(tLatAmb)
	if lAmbE != nil {
		log.Println(lAmbE)
		return
	}
	log.Println("Read", len(lAmb), "ambiguous lattices from", input)
	log.Println("Converting lattice format to internal structure")
	goldAmbLat := Lattice.Lattice2SentenceCorpus(lAmb)

	log.Println("Combining into a single gold morph graph with lattices")
	combined, missingGold := CombineTrainingInputs(goldConll, goldDisLat, goldAmbLat)

	log.Println("Combined", len(combined), "graphs, with", missingGold, "missing at least one gold path in lattice")

	log.Println("Parsing with gold to get training sequences")
	goldSequences := TrainingSequences(combined, RICH_FEATURES)
	log.Println("Generated", len(goldSequences), "training sequences")
	// Util.LogMemory()
	log.Println("Training", Iterations, "iteration(s)")
	model := Train(goldSequences, Iterations, BeamSize, RICH_FEATURES, modelFile)
	log.Println("Done Training")
	// Util.LogMemory()

	log.Println("Writing final model to", modelFile)
	WriteModel(model, modelFile)

	log.Print("Parsing test")

	log.Println("Reading ambiguous lattices from", input)
	lAmb, lAmbE = Lattice.ReadFile(input)
	if lAmbE != nil {
		log.Println(lAmbE)
		return
	}

	log.Println("Read", len(lAmb), "ambiguous lattices from", input)
	log.Println("Converting lattice format to internal structure")
	predAmbLat := Lattice.Lattice2SentenceCorpus(lAmb)

	parsedGraphs := Parse(predAmbLat, BeamSize, Dependency.ParameterModel(&PerceptronModel{model}), RICH_FEATURES)

	log.Println("Converting", len(parsedGraphs), "to conll")
	graphAsConll := Conll.MorphGraph2ConllCorpus(parsedGraphs)
	log.Println("Writing to output file")
	Conll.WriteFile(outLat, graphAsConll)
	log.Println("Wrote", len(graphAsConll), "in conll format to", outLat)

	log.Println("Writing to segmentation file")
	Segmentation.WriteFile(outSeg, parsedGraphs)
	log.Println("Wrote", len(parsedGraphs), "in segmentation format to", outSeg)

	log.Println("Writing to gold segmentation file")
	Segmentation.WriteFile(tSeg, ToMorphGraphs(combined))
	log.Println("Wrote", len(combined), "in segmentation format to", tSeg)
}

func ToMorphGraphs(graphs []*Morph.BasicMorphGraph) []NLP.MorphDependencyGraph {
	morphs := make([]NLP.MorphDependencyGraph, len(graphs))
	for i, g := range graphs {
		morphs[i] = NLP.MorphDependencyGraph(g)
	}
	return morphs
}

func MorphCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       MorphTrainAndParse,
		UsageLine: "morph [options]",
		Short:     "runs morpho-syntactic training and parsing",
		Long: `
runs morpho-syntactic training and parsing

ex:
	$ ./chukuparser morph [options]
`,
		Flag: *flag.NewFlagSet("morph", flag.ExitOnError),
	}
	cmd.Flag.IntVar(&Iterations, "it", 1, "Number of Perceptron Iterations")
	cmd.Flag.IntVar(&BeamSize, "b", 4, "Beam Size")
	cmd.Flag.StringVar(&modelFile, "m", "model", "Prefix for model file ({m}.b{b}.i{i}.model)")

	cmd.Flag.StringVar(&tConll, "tc", "", "Training Conll File")
	cmd.Flag.StringVar(&tLatDis, "td", "", "Training Disambiguated Lattices File")
	cmd.Flag.StringVar(&tLatAmb, "tl", "", "Training Ambiguous Lattices File")
	cmd.Flag.StringVar(&input, "in", "", "Test Ambiguous Lattices File")
	cmd.Flag.StringVar(&outLat, "oc", "", "Output Conll File")
	cmd.Flag.StringVar(&outSeg, "os", "", "Output Segmentation File")
	cmd.Flag.StringVar(&tSeg, "ots", "", "Output Training Segmentation File")
	return cmd
}
