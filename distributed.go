//+build cgo

package ffi

import (
	"github.com/mfdzh/filecoin-ffi/generated"
	"github.com/pkg/errors"

	spproof "fil_integrate/build/proof"
	"fil_integrate/build/state-types/abi"
)

type FallbackChallenges struct {
	Sectors    []abi.SectorNumber
	Challenges map[abi.SectorNumber][]uint64
}

type VanillaProof []byte

// GenerateWinningPoStSectorChallenge
func GeneratePoStFallbackSectorChallenges(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	sectorIds []abi.SectorNumber,
) (*FallbackChallenges, error) {
	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}

	pp, err := toFilRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	secIds := make([]uint64, len(sectorIds))
	for i, sid := range sectorIds {
		secIds[i] = uint64(sid)
	}

	resp := generated.FilGenerateFallbackSectorChallenges(
		pp, to32ByteArray(randomness), secIds, uint(len(secIds)),
		proverID,
	)
	resp.Deref()
	resp.IdsPtr = resp.IdsPtr[:resp.IdsLen]
	resp.ChallengesPtr = resp.ChallengesPtr[:resp.ChallengesLen]

	defer generated.FilDestroyGenerateFallbackSectorChallengesResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	// copy from C memory space to Go

	var out FallbackChallenges
	out.Sectors = make([]abi.SectorNumber, resp.IdsLen)
	out.Challenges = make(map[abi.SectorNumber][]uint64)
	stride := int(resp.ChallengesStride)
	for idx := range resp.IdsPtr {
		secNum := abi.SectorNumber(resp.IdsPtr[idx])
		out.Sectors[idx] = secNum
		out.Challenges[secNum] = append([]uint64{}, resp.ChallengesPtr[idx*stride:(idx+1)*stride]...)
	}

	return &out, nil
}

func GenerateSingleVanillaProof(
	replica PrivateSectorInfo,
	challange []uint64,
) ([]byte, error) {

	rep, free, err := toFilPrivateReplicaInfo(replica)
	if err != nil {
		return nil, err
	}
	defer free()

	resp := generated.FilGenerateSingleVanillaProof(rep, challange, uint(len(challange)))
	resp.Deref()
	defer generated.FilDestroyGenerateSingleVanillaProofResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	resp.VanillaProof.Deref()

	return copyBytes(resp.VanillaProof.ProofPtr, resp.VanillaProof.ProofLen), nil
}

func GenerateWinningPoStWithVanilla(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	proofs [][]byte,
) ([]spproof.PoStProof, error) {
	pp, err := toFilRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}
	fproofs, discard := toVanillaProofs(proofs)
	defer discard()

	resp := generated.FilGenerateWinningPostWithVanilla(
		pp,
		to32ByteArray(randomness),
		proverID,
		fproofs, uint(len(proofs)),
	)
	resp.Deref()
	resp.ProofsPtr = make([]generated.FilPoStProof, resp.ProofsLen)
	resp.Deref()

	defer generated.FilDestroyGenerateWinningPostResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	out, err := fromFilPoStProofs(resp.ProofsPtr)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GenerateWindowPoStWithVanilla(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	proofs [][]byte,
) (spproof.PoStProof, error) {
	pp, err := toFilRegisteredPoStProof(proofType)
	if err != nil {
		return spproof.PoStProof{}, err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return spproof.PoStProof{}, err
	}
	fproofs, discard := toVanillaProofs(proofs)
	defer discard()

	resp := generated.FilGenerateWindowPostWithVanilla(
		pp,
		to32ByteArray(randomness),
		proverID,
		fproofs, uint(len(proofs)),
	)
	resp.Deref()

	defer generated.FilDestroyGenerateWindowPostResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return spproof.PoStProof{}, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	out, err := fromFilPoStProof(resp.Proof)
	if err != nil {
		return spproof.PoStProof{}, err
	}

	return out, nil
}

type PartitionProof spproof.PoStProof

func GenerateSinglePartitionWindowPoStWithVanilla(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	proofs [][]byte,
	partitionIndex uint,
) (*PartitionProof, error) {
	pp, err := toFilRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}
	fproofs, discard := toVanillaProofs(proofs)
	defer discard()

	resp := generated.FilGenerateSingleWindowPostWithVanilla(
		pp,
		to32ByteArray(randomness),
		proverID,
		fproofs, uint(len(proofs)),
		partitionIndex,
	)
	resp.Deref()

	defer generated.FilDestroyGenerateSingleWindowPostWithVanillaResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	dpp, err := fromFilRegisteredPoStProof(resp.PartitionProof.RegisteredProof)
	if err != nil {
		return nil, err
	}

	out := PartitionProof{
		PoStProof:  dpp,
		ProofBytes: copyBytes(resp.PartitionProof.ProofPtr, resp.PartitionProof.ProofLen),
	}

	return &out, nil
}

func MergeWindowPoStPartitionProofs(
	proofType abi.RegisteredPoStProof,
	partitionProofs []PartitionProof,
) (*spproof.PoStProof, error) {
	pp, err := toFilRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	fproofs, discard, err := toPartitionProofs(partitionProofs)
	defer discard()
	if err != nil {
		return nil, err
	}

	resp := generated.FilMergeWindowPostPartitionProofs(
		pp,
		fproofs, uint(len(fproofs)),
	)
	resp.Deref()

	defer generated.FilDestroyMergeWindowPostPartitionProofsResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	dpp, err := fromFilRegisteredPoStProof(resp.Proof.RegisteredProof)
	if err != nil {
		return nil, err
	}

	out := spproof.PoStProof{
		PoStProof:  dpp,
		ProofBytes: copyBytes(resp.Proof.ProofPtr, resp.Proof.ProofLen),
	}

	return &out, nil
}
