import * as Icons from "./Icons.tsx";

export default function NavigationBar(
  props: { active: string; class?: string },
) {
  const items = [
    {
      name: "Home",
      href: "/",
    },
    {
      name: "Wie funktioniert das?",
      href: "/how-does-it-work",
    },
    {
      name: "Über",
      href: "/about",
    },
  ];

  return (
    <nav class={"flex " + props.class ?? ""}>
      <ul class="flex justify-center items-center gap-4 mx-4 my-6 flex-wrap">
        {items.map((item) => (
          <li>
            <a
              href={item.href}
              class="p-2 text-white"
            >
              {item.name}
            </a>
          </li>
        ))}

        <li class="flex items-center">
          <a
            href="https://github.com/niki2k1/kommtkevinonline"
            class="hover:text-slate-400 inline-block"
          >
            <Icons.GitHub />
          </a>
        </li>
      </ul>
    </nav>
  );
}
