import { Flex, Box, Image, Stack, Spacer } from "@chakra-ui/react";
import { NavLink as RouterLink } from "react-router-dom";
import { ColorModeButton, useColorModeValue } from "./ui/color-mode";

const Navbar = () => {
  const logoSrc = useColorModeValue(
    "/assets/logo/doq-wordmark-on-light.svg",
    "/assets/logo/doq-wordmark-on-dark.svg",
  );

  return (
    <header>
      <Flex padding="10px" alignItems="center">
        <Box>
          <RouterLink to="/" aria-label="DOQ home">
            <Image src={logoSrc} alt="DOQ" height="24px" />
          </RouterLink>
        </Box>

        <Stack
          direction={{ base: "column", md: "row" }}
          gap={5}
          marginLeft={30}
          marginRight={30}
        >
          <RouterLink
            to="/"
            className={({ isActive, isPending }) =>
              isPending
                ? "navigation pending"
                : isActive
                  ? "navigation active"
                  : "navigation"
            }
          >
            Queues
          </RouterLink>
          <RouterLink
            to="/servers"
            className={({ isActive, isPending }) =>
              isPending
                ? "navigation pending"
                : isActive
                  ? "navigation active"
                  : "navigation"
            }
          >
            Servers
          </RouterLink>
        </Stack>

        <Spacer />

        <Flex marginLeft="auto">
          <ColorModeButton />
        </Flex>
      </Flex>
    </header>
  );
};

export default Navbar;
