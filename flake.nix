{
  description = "Punchcard - A simple time tracking application";

  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";
      version = builtins.substring 0 8 lastModifiedDate;
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          punchcard = pkgs.buildGoModule {
            pname = "punchcard";
            inherit version;
            src = ./.;

            vendorHash = "sha256-lYIywCIvAwmWK5wYMM8RmeegyMNUCHZN0JrjB3szLxQ=";

            meta = with pkgs.lib; {
              description = "A simple time tracking application";
              homepage = "https://github.com/pinpox/punchcard";
              license = licenses.mit;
              mainProgram = "punchcard";
            };
          };
        });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [ go gopls gotools sqlite ];
          };
        });

      defaultPackage = forAllSystems (system: self.packages.${system}.punchcard);
    };
}
