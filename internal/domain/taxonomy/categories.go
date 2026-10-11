package taxonomy

// allCategories is the fixed list of 22 categories (PRD Appendix A). It is the
// only definition of the category set; hubs, forms and storage all read it.
var allCategories = []Category{
	{Slug: "business", Name: "Business"},
	{Slug: "health", Name: "Health"},
	{Slug: "education", Name: "Education"},
	{Slug: "housing", Name: "Housing"},
	{Slug: "finance", Name: "Finance"},
	{Slug: "travel", Name: "Travel"},
	{Slug: "tech", Name: "Tech"},
	{Slug: "relationships", Name: "Relationships"},
	{Slug: "food", Name: "Food"},
	{Slug: "government", Name: "Government"},
	{Slug: "sports", Name: "Sports"},
	{Slug: "entertainment", Name: "Entertainment"},
	{Slug: "gaming", Name: "Gaming"},
	{Slug: "parenting", Name: "Parenting"},
	{Slug: "pets", Name: "Pets"},
	{Slug: "fitness", Name: "Fitness"},
	{Slug: "fashion", Name: "Fashion"},
	{Slug: "transportation", Name: "Transportation"},
	{Slug: "environment", Name: "Environment"},
	{Slug: "career", Name: "Career"},
	{Slug: "shopping", Name: "Shopping"},
	{Slug: "other", Name: "Other"},
}
