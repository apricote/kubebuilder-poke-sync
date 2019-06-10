package pokeapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/velovix/snoreslacks/pokeapi"
)

const (
	pokeAPIHost = "https://pokeapi.co/api/v2/pokemon/"
)

type Pokemon struct {
	pokeapi.Pokemon
}

func GetPokemon(ctx context.Context, name string) (Pokemon, error) {
	res, err := http.Get(pokeAPIHost + name)
	if err != nil {
		return Pokemon{}, err
	}

	pokemon := Pokemon{}
	err = json.NewDecoder(res.Body).Decode(&pokemon)
	if err != nil {
		return Pokemon{}, err
	}

	return pokemon, nil
}
