{
  description = "Punchcard - A simple time tracking application";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          punchcard = pkgs.buildGoModule {
            pname = "punchcard";
            version = "0.1.0";
            src = ./.;
            vendorHash = "sha256-lYIywCIvAwmWK5wYMM8RmeegyMNUCHZN0JrjB3szLxQ=";

            # Embed static assets and templates
            postInstall = ''
              mkdir -p $out/share/punchcard
              cp -r ${./static} $out/share/punchcard/static
              cp -r ${./templates} $out/share/punchcard/templates
            '';

            meta = with pkgs.lib; {
              description = "A simple time tracking application";
              homepage = "https://github.com/pinpox/punchcard";
              license = licenses.mit;
              mainProgram = "punchcard";
            };
          };

          default = self.packages.${system}.punchcard;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            sqlite
          ];
        };
      }
    );
}
