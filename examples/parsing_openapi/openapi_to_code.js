export default gen((schema) =>
    render({
        "models.py": generateModels(schema),
        "routers/auth.py": generateAuthRoutes(schema),
    }),
);
