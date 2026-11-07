package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"latihan-fiber/app/model"
)

const MaxProjectsPerPortfolio = 20

var (
	ErrProjectLimit    = errors.New("jumlah proyek sudah mencapai batas maksimal 20")
	ErrProjectNotFound = errors.New("proyek tidak ditemukan di dalam portofolio")
)

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	FindByStudentID(ctx context.Context, studentID int) (model.Portfolio, error)
	Upsert(ctx context.Context, studentID int, req model.UpsertPortfolioRequest) (model.Portfolio, error)
	AddProject(ctx context.Context, studentID int, proj model.Project) (model.Portfolio, error)
	RemoveProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error)
	TopTechnologies(ctx context.Context, limit int) ([]model.TechStat, error)
}

type portfolioDocument struct {
	ID           bson.ObjectID         `bson:"_id,omitempty"`
	StudentID    int                   `bson:"student_id"`
	Headline     string                `bson:"headline"`
	Bio          string                `bson:"bio"`
	Projects     []projectDocument     `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
	CreatedAt    time.Time             `bson:"created_at"`
	UpdatedAt    time.Time             `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
	CreatedAt    time.Time     `bson:"created_at"`
}

type certificateDocument struct {
	Name   string `bson:"name"`
	Issuer string `bson:"issuer"`
	Date   string `bson:"date"`
}

func (d portfolioDocument) toModel() model.Portfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		techs := p.Technologies
		if techs == nil {
			techs = []string{}
		}
		projects = append(projects, model.Project{
			ID:           p.ID.Hex(),
			Title:        p.Title,
			Technologies: techs,
			Link:         p.Link,
			Year:         p.Year,
			CreatedAt:    p.CreatedAt,
		})
	}

	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{
			Name:   c.Name,
			Issuer: c.Issuer,
			Date:   c.Date,
		})
	}

	return model.Portfolio{
		ID:           d.ID.Hex(),
		StudentID:    d.StudentID,
		Headline:     d.Headline,
		Bio:          d.Bio,
		Projects:     projects,
		Certificates: certs,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

type portfolioMongoRepository struct {
	coll *mongo.Collection
}

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("student_unique"),
		},
		{
			Keys:    bson.D{{Key: "projects.technologies", Value: 1}},
			Options: options.Index().SetName("project_technologies_idx"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func (r *portfolioMongoRepository) FindByStudentID(ctx context.Context, studentID int) (model.Portfolio, error) {
	var doc portfolioDocument
	err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrNotFound
		}
		return model.Portfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

func (r *portfolioMongoRepository) Upsert(ctx context.Context, studentID int, req model.UpsertPortfolioRequest) (model.Portfolio, error) {
	t := now()
	certs := make([]certificateDocument, 0, len(req.Certificates))
	for _, c := range req.Certificates {
		certs = append(certs, certificateDocument{
			Name:   c.Name,
			Issuer: c.Issuer,
			Date:   c.Date,
		})
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID},
		bson.M{
			"$set": bson.M{
				"headline":     req.Headline,
				"bio":          req.Bio,
				"certificates": certs,
				"updated_at":   t,
			},
			"$setOnInsert": bson.M{
				"_id":        bson.NewObjectID(),
				"student_id": studentID,
				"projects":   []projectDocument{},
				"created_at": t,
			},
		},
		opts,
	).Decode(&updated)
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("menyimpan portofolio: %w", err)
	}
	return updated.toModel(), nil
}

func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, proj model.Project) (model.Portfolio, error) {
	t := now()
	doc := projectDocument{
		ID:           bson.NewObjectID(),
		Title:        proj.Title,
		Technologies: proj.Technologies,
		Link:         proj.Link,
		Year:         proj.Year,
		CreatedAt:    t,
	}

	limitField := "projects." + strconv.Itoa(MaxProjectsPerPortfolio-1) // "projects.19"
	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{
			"student_id": studentID,
			limitField:   bson.M{"$exists": false},
		},
		bson.M{
			"$push": bson.M{"projects": doc},
			"$set":  bson.M{"updated_at": t},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, fmt.Errorf("menambah proyek: %w", err)
	}

	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.Portfolio{}, ErrNotFound
	}
	return model.Portfolio{}, ErrProjectLimit
}

func (r *portfolioMongoRepository) RemoveProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error) {
	pid, err := parseID(projectID)
	if err != nil {
		return model.Portfolio{}, err
	}

	var updated portfolioDocument
	err = r.coll.FindOneAndUpdate(ctx,
		bson.M{
			"student_id":   studentID,
			"projects._id": pid,
		},
		bson.M{
			"$pull": bson.M{"projects": bson.M{"_id": pid}},
			"$set":  bson.M{"updated_at": now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, fmt.Errorf("menghapus proyek: %w", err)
	}

	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.Portfolio{}, ErrNotFound
	}
	return model.Portfolio{}, ErrProjectNotFound
}

func (r *portfolioMongoRepository) TopTechnologies(ctx context.Context, limit int) ([]model.TechStat, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"total": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}, {Key: "_id", Value: 1}}}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("agregasi teknologi terpopuler: %w", err)
	}
	defer cursor.Close(ctx)

	var raw []struct {
		ID    string `bson:"_id"`
		Total int    `bson:"total"`
	}
	if err := cursor.All(ctx, &raw); err != nil {
		return nil, fmt.Errorf("membaca hasil agregasi: %w", err)
	}

	stats := make([]model.TechStat, 0, len(raw))
	for _, s := range raw {
		stats = append(stats, model.TechStat{
			Technology: s.ID,
			Total:      s.Total,
		})
	}
	return stats, nil
}