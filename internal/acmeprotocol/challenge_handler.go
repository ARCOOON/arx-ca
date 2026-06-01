package acmeprotocol

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	stepacme "github.com/smallstep/certificates/acme"
	"github.com/smallstep/certificates/api/render"
)

var defaultValidator = NewValidator()

// GetChallenge handles POST /challenge/{authzID}/{chID} using arx-ca challenge validators.
func GetChallenge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := stepacme.MustDatabaseFromContext(ctx)
	linker := stepacme.MustLinkerFromContext(ctx)

	acc, err := accountFromContext(ctx)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	azID := chi.URLParam(r, "authzID")
	ch, err := db.GetChallenge(ctx, chi.URLParam(r, "chID"), azID)
	if err != nil {
		render.Error(w, r, stepacme.WrapErrorISE(err, "error retrieving challenge"))
		return
	}
	ch.AuthorizationID = azID
	if acc.ID != ch.AccountID {
		render.Error(w, r, stepacme.NewError(stepacme.ErrorUnauthorizedType,
			"account '%s' does not own challenge '%s'", acc.ID, ch.ID))
		return
	}

	jwk, err := jwkFromContext(ctx)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	if err := defaultValidator.ValidateChallenge(ctx, db, ch, jwk); err != nil {
		render.Error(w, r, stepacme.WrapErrorISE(err, "error validating challenge"))
		return
	}

	if ch.Status == stepacme.StatusValid {
		az, azErr := db.GetAuthorization(ctx, azID)
		if azErr == nil {
			_ = az.UpdateStatus(ctx, db)
		}
	}

	linker.LinkChallenge(ctx, ch, azID)

	w.Header().Add("Link", fmt.Sprintf("<%s>;rel=%q", linker.GetLink(ctx, stepacme.AuthzLinkType, azID), "up"))
	w.Header().Set("Location", linker.GetLink(ctx, stepacme.ChallengeLinkType, azID, ch.ID))
	render.JSON(w, r, ch)
}
