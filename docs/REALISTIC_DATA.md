# Realistic Data Generation

Loki includes a powerful faker system that generates realistic mock data based on field names, types, and custom configurations.

## Features

### Automatic Format Detection
Loki automatically generates realistic data for standard OpenAPI formats:

```yaml
properties:
  email:
    type: string
    format: email  # Generates: john.doe123@gmail.com
  website:
    type: string
    format: uri    # Generates: https://example.io
  id:
    type: string
    format: uuid   # Generates: 550e8400-e29b-41d4-a716-446655440000
```

### Field Name Mapping
Configure specific field names to use realistic generators in your chaos config:

```yaml
field_mapping:
  name: "person.name"        # Real names: "Sarah Johnson"
  breed: "animal.breed"      # Dog/cat breeds: "Golden Retriever"
  email: "internet.email"    # Emails: "user@example.com"
  phone: "phone.number"      # Phone numbers: "+1-555-123-4567"
  street: "address.street"   # Streets: "123 Main St"
  city: "address.city"       # Cities: "New York"
  company: "company.name"    # Companies: "Tech Solutions Inc"
```

### OpenAPI x-faker Extension
Use the `x-faker` extension in your OpenAPI spec for fine-grained control:

```yaml
properties:
  fullName:
    type: string
    x-faker: "person.name"
  jobTitle:
    type: string
    x-faker: "company.jobtitle"
  bio:
    type: string
    x-faker: "lorem.paragraph"
```

## Available Faker Types

### Person
- `person.name` / `name` - Full names: "James Smith"
- `person.firstname` / `firstname` - First names: "Emma"
- `person.lastname` / `lastname` - Last names: "Johnson"
- `internet.username` / `username` - Usernames: "johndoe123"

### Contact
- `internet.email` / `email` - Email addresses
- `phone.number` / `phone` - Phone numbers with country code

### Internet
- `internet.url` / `url` - URLs
- `internet.domain` / `domain` - Domain names
- `internet.ipv4` / `ipv4` - IPv4 addresses
- `internet.ipv6` / `ipv6` - IPv6 addresses
- `internet.useragent` / `useragent` - Browser user agents

### IDs
- `uuid` / `uuid.v4` - UUID v4
- `nanoid` - NanoID (21 characters)

### Address
- `address.street` / `street` - Street addresses
- `address.city` / `city` - City names
- `address.state` / `state` - State/province names
- `address.country` / `country` - Country names
- `address.zipcode` / `zipcode` - ZIP/postal codes

### Company
- `company.name` / `company` - Company names
- `company.jobtitle` / `jobtitle` - Job titles

### Product
- `product.name` / `product` - Product names
- `product.description` / `productdescription` - Product descriptions
- `product.category` / `category` - Product categories
- `commerce.price` / `price` - Prices with currency

### Lorem
- `lorem.word` / `word` - Single word
- `lorem.sentence` / `sentence` - Sentence (5-15 words)
- `lorem.paragraph` / `paragraph` - Paragraph (3-6 sentences)

### Animals
- `animal.type` / `animal` - Animal types: "Dog", "Cat"
- `animal.pet` / `pet` - Pet names: "Max", "Bella"
- `animal.breed` / `breed` - Pet breeds

## Configuration Examples

### Basic Field Mapping

```yaml
# chaos-config.yaml
field_mapping:
  name: "person.name"
  email: "internet.email"
  description: "lorem.sentence"

scenarios:
  # ... your chaos scenarios
```

### Type Mapping

Map OpenAPI formats to faker types:

```yaml
type_mapping:
  email: "internet.email"
  uri: "internet.url"
  hostname: "internet.domain"
```

### Complete Example

```yaml
name: "Realistic Data Config"
description: "Generate realistic mock data"

field_mapping:
  # User fields
  name: "person.name"
  username: "internet.username"
  email: "internet.email"
  phone: "phone.number"
  
  # Address fields
  street: "address.street"
  city: "address.city"
  state: "address.state"
  country: "address.country"
  zipcode: "address.zipcode"
  
  # Business fields
  company: "company.name"
  jobTitle: "company.jobtitle"
  website: "internet.url"
  
  # Content fields
  title: "lorem.word"
  description: "lorem.sentence"
  bio: "lorem.paragraph"
  
  # Product fields
  product: "product.name"
  category: "product.category"
  price: "commerce.price"
  
  # Pet fields (for petstore)
  breed: "animal.breed"
  petName: "animal.pet"

type_mapping:
  email: "internet.email"
  uri: "internet.url"
  hostname: "internet.domain"
  uuid: "uuid"

scenarios:
  - name: basic_latency
    enabled: true
    triggers:
      - paths: ["/*"]
        probability: 0.3
    chaos:
      latency:
        min: "10ms"
        max: "100ms"
```

## Usage

Start Loki with your realistic data configuration:

```bash
loki serve api-spec.yaml --chaos realistic-config.yaml
```

### Example Output

**Without realistic data:**
```json
{
  "id": 123,
  "name": "dXp3vZ1xJKL",
  "email": "user789@example.com",
  "breed": "xR2pQwE"
}
```

**With realistic data:**
```json
{
  "id": 123,
  "name": "Sarah Johnson",
  "email": "sarahjohnson456@gmail.com",
  "breed": "Golden Retriever"
}
```

## Extending with Custom Generators

The faker system is designed to be extensible. You can request new faker types by:

1. Opening an issue with your use case
2. Submitting a pull request with new generator types
3. Following the existing pattern in `internal/generator/faker.go`

### Adding a New Generator

```go
// In faker.go
func (f *Faker) CustomType() string {
    // Your generation logic
    return "generated value"
}

// In Generate() method
case "custom", "custom.type":
    return f.CustomType()
```

## Best Practices

1. **Use field mapping for domain-specific fields**: Map common field names like `name`, `email`, `address` to appropriate faker types

2. **Use x-faker for spec-specific needs**: When your OpenAPI spec has unique field requirements, use the `x-faker` extension

3. **Combine with chaos scenarios**: Realistic data works seamlessly with chaos engineering features like latency injection and error simulation

4. **Test with realistic data**: Use realistic data configurations in your development and testing environments to better simulate production

5. **Version your configs**: Keep your realistic data configurations in version control alongside your OpenAPI specs

## Troubleshooting

### Data still looks random

Make sure your chaos config is loaded:
```bash
loki serve spec.yaml --chaos config.yaml  # ✅ Correct
loki serve spec.yaml                       # ❌ No realistic data
```

### Field mapping not working

Check that:
1. The field name in your mapping matches the OpenAPI schema property name exactly
2. The field is a string type (faker only works with strings)
3. The faker type is valid (see available types above)

### Want even more realistic data

Consider:
1. Adding more specific field mappings
2. Using the `x-faker` extension in your OpenAPI spec
3. Requesting new faker types that match your domain
