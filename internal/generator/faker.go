package generator

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Faker provides realistic data generation
type Faker struct {
	rand *rand.Rand
}

// NewFaker creates a new Faker instance
func NewFaker() *Faker {
	return &Faker{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate generates realistic data based on the faker type
func (f *Faker) Generate(fakerType string) string {
	switch strings.ToLower(fakerType) {
	// Person
	case "name", "person.name", "person.fullname":
		return f.Name()
	case "firstname", "person.firstname":
		return f.FirstName()
	case "lastname", "person.lastname":
		return f.LastName()
	case "username", "internet.username":
		return f.Username()

	// Contact
	case "email", "internet.email":
		return f.Email()
	case "phone", "phone.number":
		return f.PhoneNumber()

	// Internet
	case "url", "internet.url":
		return f.URL()
	case "domain", "internet.domain":
		return f.DomainName()
	case "ipv4", "internet.ipv4":
		return f.IPv4()
	case "ipv6", "internet.ipv6":
		return f.IPv6()
	case "useragent", "internet.useragent":
		return f.UserAgent()

	// IDs
	case "uuid", "uuid.v4":
		return f.UUID()
	case "nanoid":
		return f.NanoID()

	// Address
	case "street", "address.street":
		return f.StreetAddress()
	case "city", "address.city":
		return f.City()
	case "state", "address.state":
		return f.State()
	case "country", "address.country":
		return f.Country()
	case "zipcode", "address.zipcode":
		return f.ZipCode()

	// Company
	case "company", "company.name":
		return f.CompanyName()
	case "jobtitle", "company.jobtitle":
		return f.JobTitle()

	// Product
	case "product", "product.name":
		return f.ProductName()
	case "productdescription", "product.description":
		return f.ProductDescription()
	case "category", "product.category":
		return f.Category()
	case "price", "commerce.price":
		return f.Price()

	// Lorem
	case "word", "lorem.word":
		return f.Word()
	case "sentence", "lorem.sentence":
		return f.Sentence()
	case "paragraph", "lorem.paragraph":
		return f.Paragraph()

	// Animals
	case "animal", "animal.type":
		return f.AnimalType()
	case "pet", "animal.pet":
		return f.PetName()
	case "breed", "animal.breed":
		return f.Breed()

	default:
		return f.Word()
	}
}

// Person names
var firstNames = []string{
	"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda",
	"William", "Elizabeth", "David", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Charles", "Karen", "Christopher", "Nancy", "Daniel", "Lisa",
	"Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra", "Donald", "Ashley",
	"Emma", "Olivia", "Ava", "Sophia", "Isabella", "Mia", "Charlotte", "Amelia",
	"Liam", "Noah", "Oliver", "Elijah", "Lucas", "Mason", "Logan", "Alexander",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas",
	"Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
	"Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson", "Walker", "Young",
	"Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
}

func (f *Faker) FirstName() string {
	return firstNames[f.rand.Intn(len(firstNames))]
}

func (f *Faker) LastName() string {
	return lastNames[f.rand.Intn(len(lastNames))]
}

func (f *Faker) Name() string {
	return fmt.Sprintf("%s %s", f.FirstName(), f.LastName())
}

func (f *Faker) Username() string {
	return fmt.Sprintf("%s%s%d",
		strings.ToLower(f.FirstName()),
		strings.ToLower(f.LastName()),
		f.rand.Intn(1000))
}

// Contact
func (f *Faker) Email() string {
	domains := []string{"gmail.com", "yahoo.com", "outlook.com", "example.com", "email.com"}
	return fmt.Sprintf("%s@%s",
		strings.ToLower(f.Username()),
		domains[f.rand.Intn(len(domains))])
}

func (f *Faker) PhoneNumber() string {
	return fmt.Sprintf("+1-%03d-%03d-%04d",
		f.rand.Intn(900)+100,
		f.rand.Intn(900)+100,
		f.rand.Intn(9000)+1000)
}

// Internet
func (f *Faker) URL() string {
	protocols := []string{"https", "http"}
	return fmt.Sprintf("%s://%s", protocols[f.rand.Intn(len(protocols))], f.DomainName())
}

func (f *Faker) DomainName() string {
	tlds := []string{"com", "org", "net", "io", "co", "dev"}
	return fmt.Sprintf("%s.%s", strings.ToLower(f.Word()), tlds[f.rand.Intn(len(tlds))])
}

func (f *Faker) IPv4() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		f.rand.Intn(256),
		f.rand.Intn(256),
		f.rand.Intn(256),
		f.rand.Intn(256))
}

func (f *Faker) IPv6() string {
	return fmt.Sprintf("%x:%x:%x:%x:%x:%x:%x:%x",
		f.rand.Intn(65536), f.rand.Intn(65536),
		f.rand.Intn(65536), f.rand.Intn(65536),
		f.rand.Intn(65536), f.rand.Intn(65536),
		f.rand.Intn(65536), f.rand.Intn(65536))
}

func (f *Faker) UserAgent() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
	}
	return agents[f.rand.Intn(len(agents))]
}

// IDs
func (f *Faker) UUID() string {
	return uuid.New().String()
}

func (f *Faker) NanoID() string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 21)
	for i := range b {
		b[i] = charset[f.rand.Intn(len(charset))]
	}
	return string(b)
}

// Address
var streets = []string{
	"Main St", "Oak Ave", "Maple Dr", "Cedar Ln", "Pine Rd",
	"Washington St", "Park Ave", "First St", "Second St", "Third St",
}

var cities = []string{
	"New York", "Los Angeles", "Chicago", "Houston", "Phoenix",
	"Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose",
	"Austin", "Jacksonville", "Fort Worth", "Columbus", "Charlotte",
}

var states = []string{
	"California", "Texas", "Florida", "New York", "Pennsylvania",
	"Illinois", "Ohio", "Georgia", "North Carolina", "Michigan",
}

var countries = []string{
	"United States", "Canada", "United Kingdom", "Germany", "France",
	"Australia", "Japan", "China", "India", "Brazil",
}

func (f *Faker) StreetAddress() string {
	return fmt.Sprintf("%d %s", f.rand.Intn(9999)+1, streets[f.rand.Intn(len(streets))])
}

func (f *Faker) City() string {
	return cities[f.rand.Intn(len(cities))]
}

func (f *Faker) State() string {
	return states[f.rand.Intn(len(states))]
}

func (f *Faker) Country() string {
	return countries[f.rand.Intn(len(countries))]
}

func (f *Faker) ZipCode() string {
	return fmt.Sprintf("%05d", f.rand.Intn(100000))
}

// Company
var companyTypes = []string{"LLC", "Inc", "Corp", "Group", "Solutions"}
var companyWords = []string{
	"Tech", "Data", "Cloud", "Digital", "Global", "Smart", "Innovative",
	"Advanced", "Future", "Modern", "Enterprise", "Professional",
}

func (f *Faker) CompanyName() string {
	return fmt.Sprintf("%s %s",
		companyWords[f.rand.Intn(len(companyWords))],
		companyTypes[f.rand.Intn(len(companyTypes))])
}

var jobTitles = []string{
	"Software Engineer", "Product Manager", "Data Scientist", "Designer",
	"Marketing Manager", "Sales Representative", "Customer Success Manager",
	"DevOps Engineer", "Business Analyst", "Project Manager",
}

func (f *Faker) JobTitle() string {
	return jobTitles[f.rand.Intn(len(jobTitles))]
}

// Product
var productAdjectives = []string{
	"Premium", "Deluxe", "Classic", "Modern", "Vintage", "Professional",
	"Essential", "Ultimate", "Advanced", "Basic", "Elite", "Standard",
}

var productNouns = []string{
	"Widget", "Gadget", "Tool", "Device", "Accessory", "Kit",
	"Set", "Bundle", "Package", "System", "Solution", "Product",
}

var categories = []string{
	"Electronics", "Clothing", "Books", "Home & Garden", "Sports",
	"Toys", "Automotive", "Beauty", "Food", "Health", "Pet Supplies",
}

func (f *Faker) ProductName() string {
	return fmt.Sprintf("%s %s",
		productAdjectives[f.rand.Intn(len(productAdjectives))],
		productNouns[f.rand.Intn(len(productNouns))])
}

func (f *Faker) ProductDescription() string {
	return fmt.Sprintf("High-quality %s perfect for everyday use. Features include durability, reliability, and excellent performance.",
		strings.ToLower(f.ProductName()))
}

func (f *Faker) Category() string {
	return categories[f.rand.Intn(len(categories))]
}

func (f *Faker) Price() string {
	return fmt.Sprintf("$%.2f", float64(f.rand.Intn(99900)+100)/100.0)
}

// Lorem
var words = []string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing",
	"elit", "sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore",
	"et", "dolore", "magna", "aliqua", "enim", "ad", "minim", "veniam",
}

func (f *Faker) Word() string {
	return words[f.rand.Intn(len(words))]
}

func (f *Faker) Sentence() string {
	length := 5 + f.rand.Intn(10)
	sentence := make([]string, length)
	for i := 0; i < length; i++ {
		sentence[i] = f.Word()
	}
	result := strings.Join(sentence, " ")
	return strings.ToUpper(result[:1]) + result[1:] + "."
}

func (f *Faker) Paragraph() string {
	sentences := 3 + f.rand.Intn(3)
	result := make([]string, sentences)
	for i := 0; i < sentences; i++ {
		result[i] = f.Sentence()
	}
	return strings.Join(result, " ")
}

// Animals
var animalTypes = []string{
	"Dog", "Cat", "Bird", "Fish", "Rabbit", "Hamster", "Guinea Pig",
	"Turtle", "Snake", "Lizard", "Ferret", "Horse", "Parrot",
}

var petNames = []string{
	"Max", "Bella", "Charlie", "Lucy", "Cooper", "Luna", "Buddy",
	"Daisy", "Rocky", "Molly", "Duke", "Sadie", "Bear", "Sophie",
	"Tucker", "Chloe", "Jack", "Lola", "Oliver", "Zoe", "Toby",
}

var dogBreeds = []string{
	"Labrador Retriever", "German Shepherd", "Golden Retriever", "Bulldog",
	"Beagle", "Poodle", "Rottweiler", "Yorkshire Terrier", "Boxer",
	"Dachshund", "Siberian Husky", "Great Dane", "Doberman", "Shih Tzu",
}

var catBreeds = []string{
	"Persian", "Maine Coon", "Siamese", "Ragdoll", "Bengal",
	"British Shorthair", "Abyssinian", "Birman", "Sphynx", "Scottish Fold",
}

func (f *Faker) AnimalType() string {
	return animalTypes[f.rand.Intn(len(animalTypes))]
}

func (f *Faker) PetName() string {
	return petNames[f.rand.Intn(len(petNames))]
}

func (f *Faker) Breed() string {
	// Mix of dog and cat breeds
	allBreeds := append(dogBreeds, catBreeds...)
	return allBreeds[f.rand.Intn(len(allBreeds))]
}
