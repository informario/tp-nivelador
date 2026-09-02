import sys

"""
uso:
$ python generate_compose.py 5
"""
def generate_compose(num_clients: int):
    lines = [
        "services:",
        "",
        "  server:",
        "    build:",
        "      context: ./services/server",
        "      dockerfile: Dockerfile",
        "    container_name: server",
        "    environment:",
        "      - PYTHONUNBUFFERED=1",
        "      - SERVER_HOST=server",
        "      - SERVER_PORT=5678",
        "",
    ]

    for i in range(num_clients):
        if i == 0:
            lines += [
                "  client_0:",
                "    <<: &client",
                "      build:",
                "        context: ./services/client",
                "        dockerfile: Dockerfile",
                "      depends_on:",
                "        - server",
                "      environment:",
                "        - SERVER_HOST=server",
                "        - SERVER_PORT=5678",
                "    container_name: client_0",
                "    environment:",
                "      - AGENCY_ID=0",
                "      - SERVER_HOST=server",
                "      - SERVER_PORT=5678",
                "",
            ]
        else:
            lines += [
                f"  client_{i}:",
                "    <<: *client",
                f"    container_name: client_{i}",
                "    environment:",
                f"      - AGENCY_ID={i}",
                "      - SERVER_HOST=server",
                "      - SERVER_PORT=5678",
                "",
            ]

    with open("docker-compose.yaml", "w") as f:
        f.write("\n".join(lines))


def main():
    if len(sys.argv) != 2:
        print("Uso: python generate_compose.py <cantidad_clientes>")
        sys.exit(1)

    try:
        num_clients = int(sys.argv[1])
    except ValueError:
        print("Error: la cantidad de clientes debe ser un número entero.")
        sys.exit(1)

    if num_clients < 1:
        print("Error: la cantidad de clientes debe ser al menos 1.")
        sys.exit(1)

    generate_compose(num_clients)
    print(f"docker-compose.yaml generado con {num_clients} cliente(s).")


if __name__ == "__main__":
    main()